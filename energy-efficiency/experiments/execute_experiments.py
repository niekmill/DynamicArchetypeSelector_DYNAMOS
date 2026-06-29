import requests
import time
import datetime
import csv
import constants
import argparse
import os

# Function to query Prometheus for energy consumption
def get_energy_consumption():
    # Query Prometheus
    response = requests.get(
        f"{constants.PROMETHEUS_URL}/api/v1/query",
        params={
            # Use range query, as we found that this was the most reliable in our thesis
            "query": constants.PROM_ENERGY_QUERY_RANGE
        },
    )
    # Parse the response JSON
    response_json = response.json()
    # print(f"Prometheus response status code: {response.status_code}")
    # print(f"Prometheus response: {response_json}")

    # Extract the energy data
    energy_data = {}
    # If the query was successful, return the results
    if response.status_code == 200:
        # Construct as readable energy data for each container
        for result in response_json['data']['result']:
            # Extract the container name
            container_name = result['metric'][constants.PROM_KEPLER_CONTAINER_LABEL]
            # Extract the actual result (value[0] is the timestamp)
            value = result['value'][1]
            energy_data[container_name] = value
        # Return result
        return energy_data

    # If request failed, return empty
    return {}

# Main function to execute the experiment
def run_experiment(archetype: str, output_dir, exp_rep):
    results = []
    # Get request URL based on used data_steward
    data_steward = constants.ARCH_DATA_STEWARDS[archetype]
    data_request_url = constants.REQUEST_URLS[data_steward]
    print(f"Using data steward: {data_steward}, URL:{data_request_url}")
    
    # Phase 1: Idle period
    # Wait idle period
    print("Waiting for idle period...")
    time.sleep(constants.IDLE_PERIOD)
    # Measure energy after idle (end_idle/start_active)
    idle_energy = get_energy_consumption()
    print(f"Idle Energy: {idle_energy} (in J)")

    print("Warmup approval (discarded)...")
    for attempt in range(3):
        try:
            r = requests.post(constants.APPROVAL_URL, json=constants.REQUEST_BODY_APPROVAL, headers=constants.HEADERS_APPROVAL, timeout=15)
            if r.status_code == 200:
                print(f"  warmup ok (attempt {attempt + 1})")
                break
            print(f"  warmup attempt {attempt + 1} returned {r.status_code}, retrying")
        except requests.exceptions.RequestException as e:
            print(f"  warmup attempt {attempt + 1} raised {type(e).__name__}, retrying")
        time.sleep(3)

    # Phase 2: Active period
    runs = {}
    # Record the start time of the active period
    active_start_time = time.time()
    # Execute the runs for this experiment (active state)
    for run in range(constants.NUM_EXP_ACTIONS):
        print(f"\nStarting action {run + 1}/{constants.NUM_EXP_ACTIONS}...")
        # Execute request approval
        response_approval = None
        for attempt in range(2):
            response_approval = requests.post(constants.APPROVAL_URL, json=constants.REQUEST_BODY_APPROVAL, headers=constants.HEADERS_APPROVAL)
            if response_approval.status_code == 200:
                break
            print(f"  approval attempt {attempt + 1} returned {response_approval.status_code}, retrying")
            time.sleep(2)
        # Extract relevant data from the response
        status_code_approval = response_approval.status_code
        execution_time_approval = response_approval.elapsed.total_seconds()
        print(f"Approval request completed with status: {status_code_approval}, execution time: {execution_time_approval}s")
        if status_code_approval != 200:
            print(f"  skipping action {run + 1}: approval failed after retries")
            runs[run] = {
                "appr_status_code": status_code_approval,
                "appr_exec_time": execution_time_approval,
                "data_status_code": 0,
                "data_req_exec_time": 0.0,
            }
            continue
        # Get job-id
        job_id = response_approval.json()["jobId"]
        print(f"Using job-id: {job_id}")

        time.sleep(10)

        # Construct data request body
        request_body = constants.INITIAL_REQUEST_BODY
        # Add job-id to request body
        request_body["requestMetadata"] = {"jobId": f"{job_id}"}
        # Execute data request
        response_data_request = requests.post(data_request_url, json=request_body, headers=constants.HEADERS)
        # Extract relevant data from the response
        status_code_data_request = response_data_request.status_code
        execution_time_data_request = response_data_request.elapsed.total_seconds()
        print(f"Data request completed with status: {status_code_data_request}, execution time: {execution_time_data_request}s")
        # For logging purposes print the request body if it fails
        if status_code_data_request != 200:
            print(f"Data request was not successful: request body for data request: {request_body}")
        
        # Save run data
        runs[run] = {
            "appr_status_code": status_code_approval,
            "appr_exec_time": execution_time_approval,
            "data_status_code": status_code_data_request,
            "data_req_exec_time": execution_time_data_request,
        }
        # Apply interval between requests (if not last run of sequence) 
        if (run + 1) != constants.NUM_EXP_ACTIONS:
            print("Waiting before next action...")
            time.sleep(7)

    # Before measuring the active energy, make sure the active period has passed for equal comparisons
    elapsed_time = time.time() - active_start_time
    # Add a few seconds to make sure a new Prometheus scrape is present
    remaining_time = (constants.ACTIVE_PERIOD + 2) - elapsed_time
    # If still time left to wait, sleep until the 2 minutes have passed
    if remaining_time > 0:
        print(f"Waiting for the remaining {remaining_time} seconds...")
        time.sleep(remaining_time)
    # Measure energy after active period (end_active) after the active period
    active_energy = get_energy_consumption()
    print(f"Active Energy: {active_energy} (in J)")

    # Extract results for this run
    results = {
        "runs": runs,
        "idle_energy": idle_energy,
        "active_energy": active_energy
    }

    # Save experiment results to files
    save_results(results, output_dir, exp_rep)


def save_results(results, output_dir, exp_rep):
    print("Saving experiment results to file...")
    
    # Ensure the output directory exists
    output_dir_exp = os.path.join(output_dir, f'exp_{(exp_rep+1)}')
    os.makedirs(output_dir_exp, exist_ok=True)

    # Save runs results to CSV
    runs_csv_file = os.path.join(output_dir_exp, "runs_results.csv")
    # Add the file
    with open(runs_csv_file, mode="w", newline="") as file:
        # Add run_nr as field and each key from the the runs list 
        fieldnames = ["run_nr"] + list(results["runs"][0].keys())
        writer = csv.DictWriter(file, fieldnames=fieldnames)
        writer.writeheader()
        total_exec_time = 0
        # For each run, write the data
        for run_nr, run_data in results["runs"].items():
            row = {"run_nr": run_nr}
            row.update(run_data)
            total_exec_time += run_data["appr_exec_time"] + run_data["data_req_exec_time"]
            writer.writerow(row)
    # Output file location that is clickable for the user
    print(f"Runs results saved to {os.path.join(os.getcwd(), runs_csv_file)}")

    # Calculate average execution times
    average_exec_time = total_exec_time / len(results["runs"])

    # Save experiment results to CSV
    experiment_csv_file = os.path.join(output_dir_exp, "full_experiment_results.csv")
    with open(experiment_csv_file, mode="w", newline="") as file:
        fieldnames = ["idle_energy_total", "active_energy_total", "total_energy_difference", "average_exec_time"]
        writer = csv.DictWriter(file, fieldnames=fieldnames)
        writer.writeheader()
        # Calculate total idle, active and difference energy consumption
        total_idle_energy = sum(float(value) for value in results["idle_energy"].values())
        total_active_energy = sum(float(value) for value in results["active_energy"].values())
        total_difference = total_active_energy - total_idle_energy
        
        # Add energy data
        writer.writerow({
            "idle_energy_total": total_idle_energy,
            "active_energy_total": total_active_energy,
            "total_energy_difference": total_difference,
            "average_exec_time": average_exec_time
        })
    # Output file location that is clickable for the user
    print(f"Full experiment results saved to {os.path.join(os.getcwd(), experiment_csv_file)}")

    # Save full active and idle energy values to a text file
    full_energy_file = os.path.join(output_dir_exp, "full_energy_values.txt")
    with open(full_energy_file, mode="w") as file:
        file.write("Idle Energy:\n")
        for container, value in results["idle_energy"].items():
            file.write(f"{container}: {value}\n")
        file.write("\nActive Energy:\n")
        for container, value in results["active_energy"].items():
            file.write(f"{container}: {value}\n")
    # Output file location that is clickable for the user
    print(f"Full energy values saved to {os.path.join(os.getcwd(), full_energy_file)}")


def format_timestamp():
    # Generate the current timestamp
    timestamp = datetime.datetime.now().strftime("%y%m%d-%H%M")
    return timestamp


if __name__ == "__main__":
    # Add argument parser
    parser = argparse.ArgumentParser(description="Run energy efficiency experiment")
    parser.add_argument("archetypes", type=str, nargs='+', choices=["ComputeToData", "DataThroughTTP"], 
                        help="The archetypes to use for the experiment (must be 'ComputeToData' and/or 'DataThroughTTP')")
    parser.add_argument("exp_reps", type=int, help="The number of times the experiment should be repeated")
    parser.add_argument("exp_name", type=str, choices=["baseline", "caching", "compression"], 
                        help="The name of the experiment (will be used in output files)")
    # Parse args
    args = parser.parse_args()

    # Execute experiment for each archetype and the number of repetitions
    for archetype in args.archetypes:
        print(f"Starting experiments for archetype: {archetype}")
        # Switch on this archetype by updating the weight
        request_body = constants.INITIAL_REQUEST_BODY_ARCH
        # Add weight to request body (setting weight of ComputeToData higher or lower than DataThroughTTP switches archetypes)
        request_body["weight"] = constants.WEIGHTS[archetype]
        print(f"Request body for archetype update: {request_body}")
        # Execute request
        response_archetype_update = requests.put(constants.UPDATE_ARCH_URL, json=request_body, headers=constants.HEADERS_APPROVAL)
        # Print result
        print(f"Switching archetype completed with status: {response_archetype_update.status_code}, execution time: {response_archetype_update.elapsed.total_seconds()}s")

        # Apply short idle period after switching archetypes
        print("Resting for short period before executing next experiments (after switching archetypes)...")
        time.sleep(30)

        # Execute experiments for the number of repetitions for this implementation (with the selected archetype)
        exp_reps = args.exp_reps
        exp_name = args.exp_name
        # Create a folder for the output of these experiment repetitions and archetypes
        output_dir = os.path.join('data', f'{exp_name}_{archetype}_{format_timestamp()}')
        # Ensure the output directory exists
        os.makedirs(output_dir, exist_ok=True)
        for exp_rep in range(exp_reps):
            # Print a new line before each experiment repetition
            print(f"\nStarting experiment repetition {exp_rep + 1}/{exp_reps} for archetype {archetype}...")
            # Run experiment with args
            run_experiment(archetype, output_dir, exp_rep)

            # Apply short rest period before the next experiment (if not last experiment of sequence) 
            if (exp_rep + 1) != exp_reps:
                print("Resting for a short period before doing next experiment repetition...")
                time.sleep(30)