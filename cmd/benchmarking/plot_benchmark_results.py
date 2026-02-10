import pandas as pd
import matplotlib.pyplot as plt
from matplotlib.backends.backend_pdf import PdfPages
import re
import argparse
from pathlib import Path

parser = argparse.ArgumentParser()

# Optional positional argument
parser.add_argument(
    "results_file",
    nargs="?",  # makes it optional
    default="benchmark_results.csv",
    help="Path to the results CSV file"
)

args = parser.parse_args()

# Set source and target paths
csv_path = args.results_file
pdf_path = str(Path(csv_path).with_suffix(".pdf"))

test_names = [
    "TestParallelBenchmarkSender",
    "TestParallelBenchmarkVerificationSenderProof",
    "TestParallelBenchmarkTransferProofGeneration",
    "TestParallelBenchmarkTransferServiceTransfer",
    "TestParallelBenchmarkValidatorTransfer",
]

df = pd.read_csv(csv_path)
last_row = df.iloc[-1]
timestamp = last_row["timestamp"]
markers = ['o', 's', '^', 'D', 'x', '*']
p95_marker = "X"   # single marker style for all p95 points

with PdfPages(pdf_path) as pdf:

    for test_name in test_names:

        pattern = re.compile(rf"{re.escape(test_name)}/(\d+)\s+tps")

        workers = []
        tps_values = []
        lat_p95_values = []
        lat_avg_values = []
        lat_std_values = []

        for col in df.columns:
            match = pattern.match(col)
            if match:
                worker = match.group(1)
                tps = last_row[col]
                lat_p95_col = f"{test_name}/{worker} lat-p95"
                lat_avg_col = f"{test_name}/{worker} lat-avg"
                lat_std_col = f"{test_name}/{worker} lat-std"
                lat_p95 = last_row[lat_p95_col]
                lat_avg = last_row[lat_avg_col]
                lat_std = last_row[lat_std_col]

                workers.append(worker)
                tps_values.append(tps)
                lat_p95_values.append(lat_p95)
                lat_avg_values.append(lat_avg)
                lat_std_values.append(lat_std)

        # Create figure with two subplots
        fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(16, 6))

        # --- Left subplot:
        worker_counts = [int(w) for w in workers]  # convert strings to integers

        ax1.plot(
            worker_counts,
            tps_values,
            marker='o',
            linestyle='-',
            color='tab:blue'
        )

        ax1.set_xlabel("Worker count")
        ax1.set_ylabel("TPS")
        ax1.set_title(f"{test_name}: TPS vs Worker Count\nLast Row ({timestamp})")
        ax1.grid(True)  # <-- grid added

        # --- Right subplot: TPS vs Latency with error bars (mean ± std) ---
        for i, worker in enumerate(workers):
            err = ax2.errorbar(
                tps_values[i],
                lat_avg_values[i],
                yerr=lat_std_values[i],
                marker=markers[i % len(markers)],
                label=f"{worker} worker(s)",
                capsize=5,
                capthick=2,
                linestyle='None'
            )
            color = err[0].get_color()

            ax2.scatter(
                tps_values[i],
                lat_p95_values[i],
                marker=p95_marker,
                color=color,
                s=80,           # size tweak so it stands out
                zorder=3,
                label=None     # don't add a second legend entry
            )

        # dummy scatter = to add ONE legend entry documenting p95 marker ---
        ax2.scatter([], [], marker=p95_marker, color="black", label="p95")
        ax2.set_title(f"{test_name}\nThroughput vs. Latency per worker count")
        ax2.set_xlabel("Throughput (TPS)")
        ax2.set_ylabel("Mean Latency [ms]")
        ax2.grid(True)
        ax2.legend()

        plt.tight_layout()
        plt.show()

        pdf.savefig(fig)
        plt.close(fig)

    print(f"Report saved as {pdf_path}")