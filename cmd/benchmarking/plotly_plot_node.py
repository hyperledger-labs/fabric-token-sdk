import re
import sys
from pathlib import Path
import pandas as pd
import plotly.express as px
import streamlit as st
st.set_page_config(
    layout="wide", page_icon=":chart_with_upwards_trend:", page_title="TPS Degradation")

# Generic benchmark line parser
SERVICE_RE = re.compile(
    r'^(?P<bench>\S+)-(?P<workers>\d+)\s+'
    r'(?P<iterations>\d+)\s+'
    r'(?P<tps>[\d.]+)\s+TPS\s+'
    r'(?P<rest>.+)$'
)

RE_PERCENTILE = re.compile(
    r'(?P<value>\d+)\s+ns/op\s+\((?P<name>p\d+)\)'
)


def ns2ms(ns):
    """Nano Seconds to Milliseconds"""
    return ns / 1e6


# ----------------------------
# PARSER
# ----------------------------
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


def parse(path: Path, regex=SERVICE_RE):
    rows = []

    for i, line in enumerate(path.read_text().splitlines()):
        m = regex.match(line.strip())
        if not m:
            if i == 9:
                from IPython import embed
                embed(colors="Neutral")
            continue

        data = m.groupdict()
        bench, *parts = data["bench"].split("/")

        p_dct = dict(p.split("=") for p in parts)

        row = {
            "bench": bench,
            "workers": int(data["workers"] or -1),
            "iterations": int(data["iterations"]),
            "tps": float(data["tps"]),
            **p_dct
        }

        # Parse latency percentiles dynamically
        if "rest" in data:
            for p in RE_PERCENTILE.finditer(data["rest"]):
                row[f"{p.group('name')} (ms)"] = ns2ms(int(p.group("value")))

        rows.append(row)

    return pd.DataFrame(rows)


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
            and not c.endswith("(ms)")   # exclude latency
        ]
        varying = [c for c in cols if bdf[c].nunique() > 1]
        fixed = [c for c in cols if bdf[c].nunique() <= 1]

        varying = [c for c in varying if c not in ignore_cols]
        fixed = [c for c in fixed if c not in ignore_cols]

        bdf = bdf.copy()

        # Build legend label from varying params
        bdf['series'] = (
            bdf[varying].astype(str).apply(
                lambda r: ', '.join(f'{k}={v}' for k, v in r.items()), axis=1)
            if varying else bench
        )

        # Aggregate numeric columns
        numeric_cols = [
            c for c in bdf.select_dtypes(include='number').columns
            if c not in ('workers',)
        ]
        agg = (
            bdf.groupby(['series', 'workers'])[numeric_cols]
            .mean()
            .reset_index()
            .sort_values('workers')
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

        # -----------------
        # LATENCY FIGURE 2
        # -----------------
        # latency_cols = [c for c in agg.columns if c.endswith('(ms)')]

        # if latency_cols:
        #     latency_df = agg.melt(
        #         id_vars=['series', 'workers'],
        #         value_vars=latency_cols,
        #         var_name='percentile',
        #         value_name='latency'
        #     )

        #     # Clean up the percentile name (p5 (ms) → p5)
        #     latency_df['percentile'] = latency_df['percentile'].str.replace(
        #         r' \(ms\)', '', regex=True
        #     )

        #     fig_lat = px.line(
        #         latency_df,
        #         x='workers',
        #         y='latency',
        #         color='series',
        #         line_dash='percentile',
        #         markers=True,
        #         title=f'{bench} - Latency{title_suffix}',
        #         labels={
        #             'workers': 'Workers (GOMAXPROCS)',
        #             'latency': 'Latency (ms)',
        #             'percentile': ''
        #         }
        #     )

        #     fig_lat.update_layout(template='plotly_white',
        #                           hovermode='x unified',
        #                           legend=dict(
        #                               title='nc / Percentile',
        #                               tracegroupgap=10
        #                           ))
        #     figs.append(fig_lat)

    return figs
# ----------------------------
# MAIN
# ----------------------------


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

        # Aggregate numeric columns per bench + workers
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
        for name, df in local_dfs.items():
            df['bench'] = name
        agg = pd.concat([agg, *local_dfs.values()], ignore_index=True)

        fig = px.line(
            agg,
            x="workers",
            y="tps",
            color="bench",
            markers=True,
            title=f"TPS (nc={nc})",
            labels={
                "workers": "Workers (GOMAXPROCS)",
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


def cli():
    if len(sys.argv) < 2:
        sys.exit(f"Usage: {sys.argv[0]} <log> [output.html]")

    path = Path(sys.argv[1])
    out = Path(sys.argv[2]) if len(sys.argv) > 2 else path.with_suffix(".html")

    df = parse(path)
    figs = make_figures(df)

    html = [
        fig.to_html(full_html=False,
                    include_plotlyjs=("cdn" if i == 0 else False))
        for i, fig in enumerate(figs)
    ]

    out.write_text("\n".join(html))
    print(f"Saved {len(figs)} plots to {out}")


if __name__ == "__main__":

    directory = Path("bench5")

    for path in directory.glob("*.txt"):
        with st.expander(f"`{path.stem.strip("res_transfer_").upper()}`"):
            if path.name.startswith("local"):
                df = parse(path, regex=LOCAL_RE)
            else:
                df = parse(path)
            for fig in make_figures(df):
                st.plotly_chart(fig)

    dfs = parse_combined({p.stem.strip("res_transfer_").upper(): parse(p).copy()
                          for p in directory.glob("*.txt") if not p.name.startswith("local")})
    figs = make_combined_figures(dfs, local_dfs={p.stem: parse(p, regex=LOCAL_RE).copy()
                                                 for p in directory.glob("*.txt") if p.name.startswith("local")})
    st.subheader("Combined TPS")
    tabs = st.tabs(list(figs.keys()))
    for fig, tab in zip(figs.values(), tabs):
        with tab:
            st.plotly_chart(fig)

#TODO Add allon snumbers
#TODO Write down average testdata file (token)
#Todo: 
