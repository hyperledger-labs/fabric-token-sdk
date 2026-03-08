import re
import sys
from pathlib import Path
from typing import Any
import pandas as pd
import plotly.express as px
import streamlit as st
st.set_page_config(
    layout="wide", page_icon=":chart_with_upwards_trend:", page_title="TPS Degradation", initial_sidebar_state="collapsed")

IGNORE_COLS = {"bench", "workers", "tps", "iterations", "ns/op"}
DEFAULT_BENCH_DIR = "bench"


def _is_number(s: str) -> bool:
    try:
        float(s)
        return True
    except ValueError:
        return False


def simple_parser(path: Path) -> pd.DataFrame:
    rows = []
    for ln in path.read_text().splitlines():
        if not ln.startswith("Benchmark"):
            continue
        first, *cols = ln.split()
        if not cols:
            continue

        bench_name, *params = first.split("/")
        if params:
            parts = params[-1].rsplit("-", 1)
            if len(parts) == 2 and parts[1].isdigit():
                last_param, workers = parts[0], int(parts[1])
            else:
                last_param, workers = params[-1], 1
            row: dict[str, Any] = {"bench": bench_name, "workers": workers}
            for p in [*params[:-1], last_param]:
                if "=" in p:
                    k, v = p.split("=", 1)
                    row[k] = v
        else:
            parts = bench_name.rsplit("-", 1)
            if len(parts) == 2 and parts[1].isdigit():
                bench_name, workers = parts[0], int(parts[1])
            else:
                workers = 1
            row = {"bench": bench_name, "workers": workers}

        row["iterations"] = int(cols.pop(0))

        i = 0
        while i < len(cols):
            pval = cols[i]
            j = i + 1
            while j < len(cols) and not _is_number(cols[j]):
                j += 1
            pname = " ".join(cols[i + 1:j])
            row[pname] = float(pval)
            i = j

        rows.append(row)

    df = pd.DataFrame(rows)
    if df.empty:
        return df

    if "TPS" in df.columns:
        df = df.rename(columns={"TPS": "tps"})
    for col in [c for c in df.columns if c.startswith("ns/op") and "(" in c]:
        label = col.split("(")[1].rstrip(")")
        df[f"{label} (ms)"] = df[col] / 1e6
        df = df.drop(columns=[col])

    return df


# --- STRUCTURED PARALLEL LOG PARSER ---

ANSI_ESCAPE_RE = re.compile(r'\x1b\[[0-9;]*m')
PARALLEL_RUN_RE = re.compile(r'=== RUN\s+(\S+)/(.+?)_with_(\d+)_workers')
PARALLEL_THROUGHPUT_RE = re.compile(r'Pure Throughput\s+([\d.]+)/s')
PARALLEL_LATENCY_RE = re.compile(
    r'(Min|P50 \(Median\)|Average|P95|P99\.9|P99|P5|Max)\s+'
    r'([\d.]+(?:ms|[n\xb5\xc2]+s|s))')


def _parse_ms(s: str) -> float:
    m = re.match(r'([\d.]+)(.*)', s)
    val, unit = float(m.group(1)), m.group(2)
    if unit == 'ms':
        return val
    if unit == 'ns':
        return val / 1e6
    if unit == 's':
        return val * 1e3
    raise ValueError(f"unknown unit: {unit}")


LATENCY_KEYS = {'P5': 'p5 (ms)', 'P95': 'p95 (ms)', 'P99': 'p99 (ms)'}


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
        if m and m.group(1) in LATENCY_KEYS:
            current[LATENCY_KEYS[m.group(1)]] = _parse_ms(m.group(2))
            continue

        m = re.match(r'Total Ops\s+(\d+)', line)
        if m:
            current['iterations'] = int(m.group(1))

    if current:
        rows.append(current)
    return pd.DataFrame(rows)


def has_multi_nc(df: pd.DataFrame) -> bool:
    return "nc" in df.columns and df["nc"].nunique() > 1


# --- PLOTTING ---


def _tps_fig(df, color_col, title):
    fig = px.line(df, x='workers', y='tps', color=color_col,
                  markers=True, title=title,
                  labels={'workers': 'Workers', 'tps': 'TPS', color_col: ''})
    fig.update_layout(template='plotly_white', hovermode='x unified')
    return fig


def _latency_fig(df, color_col, dash_col, title):
    latency_cols = [c for c in df.columns if c.endswith('(ms)')]
    if not latency_cols:
        return None
    melted = df.melt(id_vars=[color_col, 'workers'], value_vars=latency_cols,
                     var_name='percentile', value_name='latency')
    fig = px.line(melted, x='workers', y='latency',
                  color=dash_col, line_dash=color_col, markers=True, title=title,
                  labels={'workers': 'Workers', 'latency': 'Latency (ms)', dash_col: ''})
    fig.update_layout(template='plotly_white', hovermode='x unified')
    return fig


def _aggregate(df, group_cols):
    numeric = [c for c in df.select_dtypes(
        include='number').columns if c != 'workers']
    return (df.groupby(group_cols)[numeric]
              .mean().reset_index().sort_values('workers'))


def make_figures(df):
    param_cols = [
        c for c in df.columns if c not in IGNORE_COLS and not c.endswith("(ms)")]
    figs = []

    for bench, bdf in df.groupby('bench'):
        bdf = bdf.copy()
        varying = [c for c in param_cols if c in bdf and bdf[c].nunique() > 1]
        fixed = [c for c in param_cols if c in bdf and bdf[c].nunique()
                 <= 1 and bdf[c].notna().any()]

        bdf['series'] = (
            bdf[varying].astype(str).apply(
                lambda r: ', '.join(f'{k}={v}' for k, v in r.items()), axis=1)
            if varying else bench
        )

        agg = _aggregate(bdf, ['series', 'workers'])
        fixed_str = ', '.join(str(bdf[c].dropna().iloc[0]) for c in fixed)
        suffix = f' ({fixed_str})' if fixed_str else ''

        figs.append(_tps_fig(agg, 'series', f'{bench}{suffix}'))
        lat = _latency_fig(agg, 'series', 'percentile',
                           f'{bench} - Latency{suffix}')
        if lat:
            figs.append(lat)

    return figs


def parse_combined(dfs: dict[str, pd.DataFrame]):
    for name, df in dfs.items():
        df['bench'] = name
    dct = {}
    for df in dfs.values():
        for key, group in df.groupby("nc"):
            dct[key] = pd.concat(
                [dct.get(key, pd.DataFrame()), group], ignore_index=True)
    return dct


def make_combined_figures(dfs, local_dfs):
    figs = {}
    for nc, df in sorted(dfs.items(), key=lambda x: x[0]):
        agg = _aggregate(df, ["bench", "workers"])

        local_parts = []
        for name, ldf in local_dfs.items():
            ldf = ldf.copy()
            ldf['bench'] = name
            local_parts.append(ldf)
        agg = _aggregate(
            pd.concat([agg, *local_parts], ignore_index=True),
            ["bench", "workers"]
        )

        figs[nc] = _tps_fig(agg, 'bench', f'TPS (nc={nc})')
        lat = _latency_fig(agg, 'bench', 'percentile', f'Latency (nc={nc})')
        if lat:
            figs[f"{nc}_latency"] = lat

    return figs


def main():
    with st.sidebar:
        directory = st.text_input(
            "Directory", value=DEFAULT_BENCH_DIR, key="benchdir")
    directory = Path(directory)

    all_dfs = {}
    for path in sorted(directory.glob("*.txt")):
        df = simple_parser(path)
        if df.empty:
            continue
        all_dfs[path.stem] = df
        with st.expander(f"`{path.stem}`"):
            for fig in make_figures(df):
                st.plotly_chart(fig)

    multi = {n: df.copy() for n, df in all_dfs.items() if has_multi_nc(df)}
    single = {n: df.copy()
              for n, df in all_dfs.items() if not has_multi_nc(df)}

    dfs = parse_combined({
        n: df for n, df in multi.items()
    })

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


if __name__ == "__main__":
    main()
