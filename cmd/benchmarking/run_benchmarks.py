import csv
import os
import sys
import shutil
import subprocess
import re
import argparse
from datetime import datetime
from collections import defaultdict

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
results_csv = "benchmark_results.csv"
results_pdf = "benchmark_results.pdf"
results_IO  = "benchmark_IOstats.csv"

# --- Unit conversion ---
time_mult = {
    "ns": 1,
    "n": 1,
    "µs": 1_000,
    "µ": 1_000,
    "us": 1_000,
    "u": 1_000,
    "ms": 1_000_000,
    "m": 1_000_000,
    "s": 1_000_000_000,
}

def to_ms(value: float, unit: str) -> int:
    # First convert to nanoseconds, then scale down to milliseconds
    return int(value * time_mult[unit] / 1_000_000)

cpus_all = [1,2,4,8,16,32]
cpus = cpus_all
curves_all = ["FP256BN_AMCL", "BN254", "FP256BN_AMCL_MIRACL", "BLS12_381_BBS", "BLS12_381_BBS_GURVY", "BLS12_381_BBS_GURVY_FAST_RNG"]
curves = ["BLS12_381_BBS_GURVY"]

I=1

import csv
import os

def print_IOstats(d, filename):
    file_exists = os.path.exists(filename)

    with open(filename, "a", newline="") as f:
        writer = csv.writer(f)

        # Write header only once
        if not file_exists:
            writer.writerow(["bench", "bits", "I", "O", "allocs", "RAM bytes", "time ms"])

        # --- Grouped iteration ---
        for bench in sorted(d.keys()):

            if "time" not in d[bench]:
                continue

            for bits in sorted(d[bench]["time"].keys(), key=int):
                for I in sorted(d[bench]["time"][bits].keys(), key=int):
                    for O in sorted(d[bench]["time"][bits][I].keys(), key=int):

                        allocs = d[bench]["allocs"].get(bits, {}).get(I, {}).get(O, "")
                        ram    = d[bench]["ram"].get(bits, {}).get(I, {}).get(O, "")
                        time   = d[bench]["time"].get(bits, {}).get(I, {}).get(O, "")

                        writer.writerow([bench, bits, I, O, allocs, ram, time])

def run_and_parse_non_parallel_metrics(benchName, params, curve="BLS12_381_BBS_GURVY", folder=transfer_benchmarks_folder) -> dict:
    global I
    global output_folder_path
    global count, timeout

    if folder == "":
        folder = transfer_benchmarks_folder

    cmd = f"go test {folder} -run='^$' -bench={benchName} -v -benchmem -count={count} -cpu=32 -curves={curve} -timeout {timeout} {params} | tee bench.txt; benchstat bench.txt" 
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

    d = defaultdict(lambda: defaultdict(lambda: defaultdict(lambda: defaultdict(dict))))
    name = None
    time_ms = None
    ram_bytes = None
    allocs = None

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
    time_re = re.compile(
        r"^\S+"                    # benchmark name
        r".*?bits_(\d+)"           # (1) bits value
        r".*?_#i_(\d+)"            # (2) input count
        r".*?_#o_(\d+)"            # (3) output count
        r"\S*\s+"                  # rest of name until whitespace
        r"([\d.]+)\s*([a-zµ]+)"    # time: (4) number + (5) unit
    )
    time_mean_re  = re.compile(r"^geomean\s+([\d.]+)\s*([a-zµ]+)")
    ram_re = re.compile(
        r"^\S+"                    # benchmark name
        r".*?bits_(\d+)"           # (1) bits value
        r".*?_#i_(\d+)"            # (2) input count
        r".*?_#o_(\d+)"            # (3) output count
        r"\S*\s+"                  # rest of name until whitespace
        r"([\d.]+)\s*(B|Ki|Mi)"    # ram: (4) number + (5) unit
    )
    ram_mean_re   = re.compile(r"^geomean\s+([\d.]+)\s*(B|Ki|Mi)")
    alloc_re = re.compile(
        r"^\S+"                    # benchmark name
        r".*?bits_(\d+)"           # (1) bits value
        r".*?_#i_(\d+)"            # (2) input count
        r".*?_#o_(\d+)"            # (3) output count
        r"\S*\s+"                  # rest of name until whitespace
        r"([\d.]+)\s*([kM]?)"      # allocs: (4) number + (5) unit?
    )
    alloc_mean_re = re.compile(r"^geomean\s+([\d.]+)\s*([kM]?)")

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
            m = time_mean_re.match(line)
            if m:
                val, unit = m.groups()
                time_ms_mean = to_ms(float(val), unit) 

            m = time_re.match(line)
            if m:
                bits, Inum, Onum, val, unit = m.groups()
                time_ms = to_ms(float(val), unit) 
                d[benchName]["time"][bits][Inum][Onum] = time_ms

        elif section == "ram":
            m = ram_mean_re.match(line)
            if m:
                val, unit = m.groups()
                ram_bytes_mean = int(float(val) * ram_mult[unit])
            m = ram_re.match(line)
            if m:
                bits, Inum, Onum, val, unit = m.groups()
                ram_bytes = int(float(val) * ram_mult[unit])
                d[benchName]["ram"][bits][Inum][Onum] = ram_bytes

        elif section == "allocs":
            m = alloc_mean_re.match(line)
            if m:
                val, unit = m.groups()
                allocs_mean = int(float(val) * alloc_mult[unit])
            m = alloc_re.match(line)
            if m:
                bits, Inum, Onum, val, unit = m.groups()
                allocs = int(float(val) * alloc_mult[unit])
                d[benchName]["allocs"][bits][Inum][Onum] = allocs

    if not all([time_ms_mean, ram_bytes_mean, allocs_mean is not None]):
        raise ValueError("Failed to parse benchmark output")

    name = benchName
    print_IOstats(d, results_IO)
    print(f"{benchName} results:")
    print(f"time (ms)   : {time_ms_mean}")
    print(f"RAM  (bytes): {ram_bytes_mean}")
    print(f"allocs      : {allocs_mean}")
    return {
        f"{name} time": time_ms_mean,
        f"{name} RAM": ram_bytes_mean,
        f"{name} allocs": allocs_mean,
    }

def run_and_parse_parallel_metrics(benchName, params, cpu=1, curve="BLS12_381_BBS_GURVY", folder=transfer_benchmarks_folder) -> dict:
    if folder == "":
        folder = transfer_benchmarks_folder

    global I
    global timeout
    global output_folder_path

    cmd = f"go test {folder} -test.run={benchName} -test.v -test.timeout {timeout} -bits='32' -num_inputs='2' -num_outputs='2' -cpu={cpu} -workers='NumCPU' -curves={curve} -duration='10s' -setup_samples=128 {params}"
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

    log_file_path = os.path.join(output_folder_path, benchName+"-"+str(cpu)+".log")
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


    # --- Regexes ---
    tps_re = re.compile(r"Real Throughput\s+([\d.]+)/s")

    lat_p95_re = re.compile(r"P95\s+([\d.]+)(ns|µs|us|ms|s)")
    lat_avg_re = re.compile(r"Average\s+([\d.]+)(ns|µs|us|ms|s)")
    lat_std_re = re.compile(r"Std Dev\s+([\d.]+)(ns|µs|us|ms|s)")

    # --- Parse values ---
    tps = float(tps_re.search(output).group(1))

    lat_p95, lat_p95_unit = lat_p95_re.search(output).groups()
    lat_avg, lat_avg_unit = lat_avg_re.search(output).groups()
    lat_std, lat_std_unit = lat_std_re.search(output).groups()

    name = benchName
    print(f"{benchName} results:")
    print(f"tps         : {tps}")
    print(f"latency (ms): p95,mean,std : {to_ms(float(lat_p95), lat_p95_unit)}, {to_ms(float(lat_avg), lat_avg_unit)}, {to_ms(float(lat_std), lat_std_unit)}")
    return {
        f"{name}/{cpu} tps": tps,
        f"{name}/{cpu} lat-p95": to_ms(float(lat_p95), lat_p95_unit),
        f"{name}/{cpu} lat-avg": to_ms(float(lat_avg), lat_avg_unit),
        f"{name}/{cpu} lat-std": to_ms(float(lat_std), lat_std_unit)
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
    # ('BenchmarkSender', "-num_inputs 1,2,3 -num_outputs 1,2,3 ", ""), 
    ('BenchmarkVerificationSenderProof', "-num_inputs 1,2,3 -num_outputs 1,2,3 -bits 32,64", ""),
    # ('BenchmarkTransferProofGeneration', "", ""), 
    # ('BenchmarkIssuer', "", issuer_benchmarks_folder), 
    # ('BenchmarkProofVerificationIssuer', "", issuer_benchmarks_folder), 
    # ('BenchmarkTransferServiceTransfer', "", v1_benchmarks_folder), 
]
parallel_tests = [
    # ('TestParallelBenchmarkSender', "", ""), 
    ('TestParallelBenchmarkVerificationSenderProof', "", ""),
    # ('TestParallelBenchmarkTransferProofGeneration', "", ""),
    # ('TestParallelBenchmarkTransferServiceTransfer', "", v1_benchmarks_folder), 
    # ('TestParallelBenchmarkValidatorTransfer', "", validator_benchmarks_folder), 
]

results = {}
print("\n*******************************************************")
print("Running non-parallel tests")

for testName, params, benchType in non_parallel_tests:
    if (benchName == "") or (benchName == testName):
        for curve in curves:
            results.update(run_and_parse_non_parallel_metrics(testName, params, curve, benchType)) 

print("\n*******************************************************")
print("Running parallel tests")
for testName, params, folder in parallel_tests:
    if (benchName == "") or (benchName == testName):
        for curve in curves:
            for cpu in cpus:
                results.update(run_and_parse_parallel_metrics(testName, params, cpu, curve, folder))

# add new row to benchmark_results.csv and copy it to the output folder
# but not if we just run a single bench as a test
if benchName == "": # we ran all the benchmarks
    append_dict_as_row(results_csv, results)
    src = os.path.join(".", results_csv)
    dst = os.path.join(output_folder_path, results_csv)
    if os.path.exists(src) and not os.path.exists(dst):
            cmd = f"python plot_benchmark_results.py" 
            print(f"Running: {cmd}")
            subprocess.run(
                cmd,
                shell=True,
                capture_output=True,
                text=True,
                check=True
            )
            shutil.copy(src, dst)
            os.remove(src)

    src = os.path.join(".", results_pdf)
    dst = os.path.join(output_folder_path, results_pdf)
    if os.path.exists(src) and not os.path.exists(dst):
        shutil.copy(src, dst)
        os.remove(src)

    src = os.path.join(".", results_IO)
    dst = os.path.join(output_folder_path, results_IO)
    if os.path.exists(src) and not os.path.exists(dst):
        shutil.copy(src, dst)
        os.remove(src)

    if os.path.exists("bench.txt"):
        os.remove("bench.txt")
