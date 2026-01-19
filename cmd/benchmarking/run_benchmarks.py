import csv
import os
import sys
import shutil
import subprocess
import re
import argparse
from datetime import datetime

parser = argparse.ArgumentParser(description="run_benchmark.py script")
parser.add_argument(
    "--count",       # use --count from the command line
    type=int,        # expect an integer
    default=10,      # default if not provided
    help="Number of repetitions (default: 10)"
)
parser.add_argument(
    "--timeout",    # use --timeout from the command line
    type=str,       # expect an integer
    default="0",      # default if not provided
    help="timeout for the run (default: 0, i.e. not timeout)"
)
parser.add_argument(
    "--benchName",  # use --benchName from the command line
    type=str,       # expect a string
    default="",     # default if not provided
    help="benchmark name to run (default is to run all benchmarks)"
)

args = parser.parse_args()
count = args.count
timeout = args.timeout
benchName = args.benchName

TOKENSDK_ROOT = os.environ.get("TOKENSDK_ROOT", "../../")
output_folder_path = ""
v1_benchmarks_folder = os.path.join(TOKENSDK_ROOT, "token/core/zkatdlog/nogh/v1")
transfer_benchmarks_folder = os.path.join(TOKENSDK_ROOT, "token/core/zkatdlog/nogh/v1/transfer")
issuer_benchmarks_folder = os.path.join(TOKENSDK_ROOT, "token/core/zkatdlog/nogh/v1/issue")
validator_benchmarks_folder = os.path.join(TOKENSDK_ROOT, "token/core/zkatdlog/nogh/v1/validator")

I=1

def run_and_parse_non_parallel_metrics(benchName, params, folder=transfer_benchmarks_folder) -> dict:
    global I
    global output_folder_path
    global count, timeout

    if folder == "":
        folder = transfer_benchmarks_folder

    cmd = f"go test {folder} -run='^$' -bench={benchName} -v -benchmem -count={count} -cpu=1 -timeout {timeout} {params} | tee bench.txt; benchstat bench.txt" 
    print(f"{I} Running: {cmd}")
    I = I+1
    result = subprocess.run(
        cmd,
        shell=True,
        capture_output=True,
        text=True,
        check=True
    )

    log_file_path = os.path.join(output_folder_path, benchName+".log")
    if not os.path.exists(log_file_path):
        with open(log_file_path, "w", encoding="utf-8") as f:
            f.write(result.stdout)

    output = result.stdout.splitlines()

    name = None
    time_ns = None
    ram_bytes = None
    allocs = None

    # --- Unit multipliers ---
    time_mult = {
        "n": 1,
        "µ": 1_000,
        "u": 1_000,
        "m": 1_000_000,
        "s": 1_000_000_000,
    }

    ram_mult = {
        "B": 1,
        "Ki": 1024,
        "Mi": 1024 ** 2,
    }

    alloc_mult = {
        "": 1,
        "k": 1_000,
        "M": 1_000_000,
    }

    # --- Regexes ---
    time_re = re.compile(r"^(\S+)\s+([\d.]+)\s*([a-zµ]+)")
    ram_re = re.compile(r"^(\S+)\s+([\d.]+)\s*(B|Ki|Mi)")
    alloc_re = re.compile(r"^(\S+)\s+([\d.]+)\s*([kM]?)")

    section = None

    for line in output:
        line = line.strip()
        if not line:
            continue

        if "sec/op" in line:
            section = "time"
            continue
        elif "B/op" in line:
            section = "ram"
            continue
        elif "allocs/op" in line:
            section = "allocs"
            continue

        if section == "time":
            m = time_re.match(line)
            if m:
                name, val, unit = m.groups()
                time_ns = int(float(val) * time_mult[unit])

        elif section == "ram":
            m = ram_re.match(line)
            if m:
                name, val, unit = m.groups()
                ram_bytes = int(float(val) * ram_mult[unit])

        elif section == "allocs":
            m = alloc_re.match(line)
            if m:
                name, val, unit = m.groups()
                allocs = int(float(val) * alloc_mult[unit])

    if not all([name, time_ns, ram_bytes, allocs is not None]):
        raise ValueError("Failed to parse benchmark output")

    name = benchName
    return {
        f"{name} time": time_ns,
        f"{name} RAM": ram_bytes,
        f"{name} allocs": allocs,
    }

def run_and_parse_parallel_metrics(benchName, params, folder=transfer_benchmarks_folder) -> dict:
    if folder == "":
        folder = transfer_benchmarks_folder

    global I
    global timeout
    global output_folder_path

    cmd = f"go test {folder} -test.run={benchName} -test.v -test.timeout {timeout} -bits='32' -num_inputs='2' -num_outputs='2' -workers='NumCPU' -duration='10s' -setup_samples=128 {params}"
    print(f"{I} Running: {cmd}")
    I = I+1

    # --- Run command ---
    result = subprocess.run(
        cmd,
        shell=True,
        capture_output=True,
        text=True,
        check=True
    )

    log_file_path = os.path.join(output_folder_path, benchName+".log")
    if not os.path.exists(log_file_path):
        with open(log_file_path, "w", encoding="utf-8") as f:
            f.write(result.stdout)

    output = result.stdout

    # --- Extract test name ---
    name_re = re.compile(
        r"=== RUN\s+([a-zA-Z0-9]+/.+?)\s*$",
        re.MULTILINE
    )
    name_match = name_re.search(output)
    if not name_match:
        raise ValueError("Could not extract test name")

    # --- Unit conversion ---
    time_mult = {
        "ns": 1,
        "µs": 1_000,
        "us": 1_000,
        "ms": 1_000_000,
        "s": 1_000_000_000,
    }

    def to_ns(value: float, unit: str) -> int:
        return int(value * time_mult[unit])

    # --- Regexes ---
    real_tp_re = re.compile(r"Real Throughput\s+([\d.]+)/s")
    pure_tp_re = re.compile(r"Pure Throughput\s+([\d.]+)/s")

    min_lat_re = re.compile(r"Min\s+([\d.]+)(ns|µs|us|ms|s)")
    avg_lat_re = re.compile(r"Average\s+([\d.]+)(ns|µs|us|ms|s)")
    max_lat_re = re.compile(r"Max\s+([\d.]+)(ns|µs|us|ms|s)")

    # --- Parse values ---
    real_tp = float(real_tp_re.search(output).group(1))
    pure_tp = float(pure_tp_re.search(output).group(1))

    min_val, min_unit = min_lat_re.search(output).groups()
    avg_val, avg_unit = avg_lat_re.search(output).groups()
    max_val, max_unit = max_lat_re.search(output).groups()

    name = benchName
    return {
        f"{name} real throughput": real_tp,
        f"{name} pure throughput": pure_tp,
        f"{name} min latency": to_ns(float(min_val), min_unit),
        f"{name} average latency": to_ns(float(avg_val), avg_unit),
        f"{name} max latency": to_ns(float(max_val), max_unit),
    }

def append_dict_as_row(filename: str, data: dict):
    file_exists = os.path.exists(filename)
    timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    row = {"timestamp": timestamp, **data}
    fieldnames = row.keys()

    with open(filename, "a", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=row.keys())

        if not file_exists:
            writer.writeheader()

        writer.writerow(row)

timestamp = datetime.now().strftime("%Y-%m-%d_%H-%M-%S")
output_folder_name = f"benchmark_logs_{timestamp}"
output_folder_path = os.path.join(".", output_folder_name)
os.makedirs(output_folder_path, exist_ok=True)

non_parallel_tests = [
    ('BenchmarkSender', "", ""),
    ('BenchmarkVerificationSenderProof', "", ""),
    ('BenchmarkTransferProofGeneration', "", ""), 
    ('BenchmarkIssuer', "", issuer_benchmarks_folder), 
    ('BenchmarkProofVerificationIssuer', "", issuer_benchmarks_folder), 
    ('BenchmarkVerificationSenderProof', "", ""), 
    ('BenchmarkTransferServiceTransfer', "", v1_benchmarks_folder), 
]
parallel_tests = [
    ('TestParallelBenchmarkSender', "", ""), 
    ('TestParallelBenchmarkVerificationSenderProof', "", ""),
    ('TestParallelBenchmarkTransferProofGeneration', "", ""),
    ('TestParallelBenchmarkTransferServiceTransfer', "", v1_benchmarks_folder), 
    ('TestParallelBenchmarkValidatorTransfer', "", validator_benchmarks_folder), 
]

results = {}
print("\n*******************************************************")
print("Running non-parallel tests")

for testName, params, benchType in non_parallel_tests:
    if (benchName == "") or (benchName == testName):
        results.update(run_and_parse_non_parallel_metrics(testName, params, benchType)) 

print("\n*******************************************************")
print("Running parallel tests")
for testName, params, folder in parallel_tests:
    if (benchName == "") or (benchName == testName):
       results.update(run_and_parse_parallel_metrics(testName, params, folder))

# add new row to benchmark_results.csv and copy it to the output folder
# but not if we just run a single bench as a test
if benchName == "": # we ran all the benchmarks
    append_dict_as_row("benchmark_results.csv", results)
    src = os.path.join(".", "benchmark_results.csv")
    dst = os.path.join(output_folder_path, "benchmark_results.csv")
    if os.path.exists(src) and not os.path.exists(dst):
        shutil.copy(src, dst)
