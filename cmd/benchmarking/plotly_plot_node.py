import re
import sys
from pathlib import Path
import pandas as pd
import plotly.express as px
import streamlit as st
st.set_page_config(
    layout="wide", page_icon=":chart_with_upwards_trend:", page_title="TPS Degradation")

DEFAULT_BENCH_DIR = "bench6"

SERVICE_RE = re.compile(
    r'^(?P<bench>\S+)-(?P<workers>\d+)\s+'
    r'(?P<iterations>\d+)\s+'
    r'(?P<tps>[\d.]+)\s+TPS\s+'
    r'(?P<rest>.+)$'
)

RE_PERCENTILE = re.compile(
    r'(?P<value>\d+)\s+ns/op\s+\((?P<name>p\d+)\)'
)

LOCAL_RE = re.compile(
    r'^'
    r'(?P<bench>Benchmark[^\s/]+)'          # Benchmark name
    r'(?:/(?P<params>.+?))?'                # everything after / (lazy)
    r'(?:-(?P<workers>\d+))?'               # trailing -workers
    r'\s+'
    r'(?P<iterations>\d+)'                  # iterations
    r'\s+'
    r'(?P<ns>\d+)\s+ns/op'                  # ns/op
    r'\s+'
    r'(?P<tps>[\d.]+)\s+TPS'                # TPS
    r'$'
)


def ns2ms(ns):
    """Nano Seconds to Milliseconds"""
    return ns / 1e6


def parse(path: Path, regex=SERVICE_RE):
    rows = []

    for line in path.read_text().splitlines():
        m = regex.match(line.strip())
        if not m:
            continue

        data = m.groupdict()
        bench, *parts = data["bench"].split("/")

        if data.get("params"):
            parts.extend(data["params"].split("/"))

        p_dct = dict(p.split("=", 1) for p in parts if "=" in p)

        row = {
            "bench": bench,
            "workers": int(data["workers"] or -1),
            "iterations": int(data["iterations"]),
            "tps": float(data["tps"]),
            **p_dct
        }

        if "rest" in data:
            for p in RE_PERCENTILE.finditer(data["rest"]):
                row[f"{p.group('name')} (ms)"] = ns2ms(int(p.group("value")))

        rows.append(row)

    return pd.DataFrame(rows)


def try_parse(path: Path):
    """Try SERVICE_RE first, fall back to LOCAL_RE."""
    df = parse(path, regex=SERVICE_RE)
    if df.empty:
        df = parse(path, regex=LOCAL_RE)
    return df


# ----------------------------
# STRUCTURED PARALLEL LOG PARSER
# ----------------------------
ANSI_ESCAPE_RE = re.compile(r'\x1b\[[0-9;]*m')

PARALLEL_RUN_RE = re.compile(
    r'=== RUN\s+(\S+)/(.+?)_with_(\d+)_workers'
)

PARALLEL_THROUGHPUT_RE = re.compile(
    r'Pure Throughput\s+([\d.]+)/s'
)

PARALLEL_LATENCY_RE = re.compile(
    r'(Min|P50 \(Median\)|Average|P95|P99\.9|P99|P5|Max)\s+'
    r'([\d.]+(?:ms|[n\xb5\xc2]+s|s))'
)


def _parse_ms(s: str) -> float:
    """Parse a duration string (e.g. '36.86ms') into milliseconds."""
    m = re.match(r'([\d.]+)(.*)', s)
    val, unit = float(m.group(1)), m.group(2)
    if unit == 'ms':
        return val
    if unit == 'ns':
        return val / 1e6
    if unit == 's':
        return val * 1e3
    raise ValueError(f"unknown unit: {unit}")


def parse_parallel_log(path: Path) -> pd.DataFrame:
    text = ANSI_ESCAPE_RE.sub('', path.read_text())

    rows: list[dict] = []
    current: dict = {}

    for line in text.splitlines():
        line = line.strip()

        m = PARALLEL_RUN_RE.match(line)
        if m:
            if current:
                rows.append(current)
            bench, params_raw, workers = m.group(
                1), m.group(2), int(m.group(3))
            current = {'bench': bench, 'workers': workers}
            setup_m = re.search(r'Setup\((.+?)\)', params_raw)
            if setup_m:
                for token in setup_m.group(1).split(',_'):
                    token = token.strip('_').lstrip('#')
                    if '_' in token:
                        k, v = token.split('_', 1)
                        current[k] = v
            continue

        m = PARALLEL_THROUGHPUT_RE.match(line)
        if m:
            current['tps'] = float(m.group(1))
            continue

        m = PARALLEL_LATENCY_RE.match(line)
        if m:
            name, raw = m.group(1), m.group(2)
            ms_val = _parse_ms(raw)
            key_map = {
                # 'P50 (Median)': 'p50 (ms)',
                'P5':           'p5 (ms)',
                'P95':          'p95 (ms)',
                'P99':          'p99 (ms)',
                # 'P99.9':        'p99.9 (ms)',
                # 'Min':          'min (ms)',
                # 'Max':          'max (ms)',
                # 'Average':      'avg (ms)',
            }
            if name in key_map:
                current[key_map[name]] = ms_val
            continue

        m = re.match(r'Total Ops\s+(\d+)', line)
        if m:
            current['iterations'] = int(m.group(1))

    if current:
        rows.append(current)

    return pd.DataFrame(rows)


def has_multi_nc(df: pd.DataFrame) -> bool:
    return "nc" in df.columns and df["nc"].nunique() > 1


# ----------------------------
# PLOTTING
# ----------------------------


def make_figures(df):

    param_cols = [c for c in df.columns if c not in (
        'bench', 'workers', 'tps')]
    ignore_cols = ["iterations"]

    figs = []

    for bench, bdf in df.groupby('bench'):

        cols = [
            c for c in param_cols
            if bdf[c].notna().any()
            and not c.endswith("(ms)")
        ]
        varying = [c for c in cols if bdf[c].nunique() > 1]
        fixed = [c for c in cols if bdf[c].nunique() <= 1]

        varying = [c for c in varying if c not in ignore_cols]
        fixed = [c for c in fixed if c not in ignore_cols]

        bdf = bdf.copy()

        # Build legend label
        bdf['series'] = (
            bdf[varying].astype(str).apply(
                lambda r: ', '.join(f'{k}={v}' for k, v in r.items()), axis=1)
            if varying else bench
        )

        numeric_cols = [
            c for c in bdf.select_dtypes(include='number').columns
            if c not in ('workers',)
        ]
        agg = (
            bdf.groupby(['series', 'workers'])[numeric_cols]
            .mean()
            .reset_index()
            .sort_values('workers', ascending=False)
        )

        fixed_str = ', '.join(f'{bdf[c].dropna().iloc[0]}' for c in fixed)
        title_suffix = f' ({fixed_str})' if fixed_str else ''

        # -----------------
        # TPS FIGURE
        # -----------------
        fig_tps = px.line(
            agg,
            x='workers',
            y='tps',
            color='series',
            markers=True,
            title=f'{bench}\n{title_suffix}',
            labels={
                'workers': 'Workers (GOMAXPROCS)',
                'tps': 'TPS',
                'series': ''
            }
        )
        fig_tps.update_layout(template='plotly_white',
                              hovermode='x unified')
        figs.append(fig_tps)

        # -----------------
        # LATENCY FIGURE
        # -----------------
        latency_cols = [c for c in agg.columns if c.endswith('(ms)')]

        if latency_cols:
            latency_df = agg.melt(
                id_vars=['series', 'workers'],
                value_vars=latency_cols,
                var_name='percentile',
                value_name='latency'
            )

            fig_lat = px.line(
                latency_df,
                x='workers',
                y='latency',
                color='percentile',
                line_dash='series',
                markers=True,
                title=f'{bench} - Latency{title_suffix}',
                labels={
                    'workers': 'Workers (GOMAXPROCS)',
                    'latency': 'Latency (ms)',
                    'percentile': ''
                }
            )

            fig_lat.update_layout(template='plotly_white',
                                  hovermode='x unified',
                                  legend=dict(tracegroupgap=10))
            figs.append(fig_lat)

    return figs


def parse_combined(dfs: dict[str, pd.DataFrame]):
    for name, df in dfs.items():
        df['bench'] = name
    dct = {}
    for df in dfs.values():
        for key, group in df.groupby("nc"):
            if key not in dct:
                dct[key] = group
            else:
                dct[key] = pd.concat([dct[key], group], ignore_index=True)
    return dct


def make_combined_figures(dfs: dict[str, pd.DataFrame],
                          local_dfs: dict[str, pd.DataFrame]):

    figs = {}

    for nc, df in sorted(dfs.items(), key=lambda x: x[0]):

        df = df.copy()

        numeric_cols = [
            c for c in df.select_dtypes(include="number").columns
            if c != "workers"
        ]

        agg = (
            df.groupby(["bench", "workers"])[numeric_cols]
            .mean()
            .reset_index()
            .sort_values("workers")
        )

        # -------------------------
        # TPS FIGURE
        # -------------------------
        agg["bench"] += "-2machines"
        local_parts = []
        for name, ldf in local_dfs.items():
            ldf = ldf.copy()
            ldf['bench'] = name
            local_parts.append(ldf)
        agg = pd.concat([agg, *local_parts],
                        ignore_index=True).sort_values(["workers", "tps"], ascending=[True, False])

        fig = px.line(
            agg,
            x="workers",
            y="tps",
            color="bench",
            markers=True,
            title=f"TPS (nc={nc})",
            labels={
                "workers": "Workers",
                "tps": "TPS",
                "bench": ""
            }
        )

        fig.update_layout(
            template="plotly_white",
            hovermode="x unified"
        )

        figs[nc] = fig

        latency_cols = [c for c in agg.columns if c.endswith("(ms)")]

        if latency_cols:
            latency_df = agg.melt(
                id_vars=["bench", "workers"],
                value_vars=latency_cols,
                var_name="percentile",
                value_name="latency"
            )
            fig_lat = px.line(
                latency_df,
                x="workers",
                y="latency",
                color="bench",
                line_dash="percentile",
                markers=True,
                title=f"Latency (nc={nc})",
                labels={
                    "workers": "Workers (GOMAXPROCS)",
                    "latency": "Latency (ms)",
                    "bench": ""
                }
            )

            figs[f"{nc}_latency"] = fig_lat

    return figs


if __name__ == "__main__":

    directory = Path(sys.argv[1] if len(sys.argv) > 1 else DEFAULT_BENCH_DIR)

    all_dfs = {}
    for path in sorted(directory.glob("*.txt")):
        df = try_parse(path)
        if df.empty:
            continue
        all_dfs[path.stem] = df
        with st.expander(f"`{path.stem.removeprefix('res_transfer_').upper()}`"):
            for fig in make_figures(df):
                st.plotly_chart(fig)

    multi = {name: df.copy()
             for name, df in all_dfs.items() if has_multi_nc(df)}
    single = {name: df.copy()
              for name, df in all_dfs.items() if not has_multi_nc(df)}

    dfs = parse_combined({
        name.removeprefix("res_transfer_").upper(): df
        for name, df in multi.items()
    })
    for path in sorted(directory.glob("*.csv")):
        df = pd.read_csv(path)
        assert not df.empty
        with st.expander(f"`{path.stem}`"):
            st.dataframe(df)
        single[path.stem] = df
    for path in sorted(directory.glob("*.log")):
        df = parse_parallel_log(path)
        assert not df.empty
        with st.expander(f"`{path.stem}`"):
            st.dataframe(df)
        single[path.stem] = df

    figs = make_combined_figures(dfs, local_dfs=single)
    st.subheader("Combined TPS")
    tabs = st.tabs(list(figs.keys()))
    for fig, tab in zip(figs.values(), tabs):
        with tab:
            st.plotly_chart(fig)
