import streamlit as st
import requests
import os
from datetime import datetime
try:
    import psycopg2
except ImportError:
    psycopg2 = None

SPRING_BOOT_URL = os.getenv("SPRING_BOOT_URL", "http://localhost:8080")
GIN_URL         = os.getenv("GIN_URL",         "http://localhost:8081")
FASTAPI_URL     = os.getenv("FASTAPI_URL",     "http://localhost:8082")

PG_HOST     = os.getenv("PG_HOST",     "localhost")
PG_USER     = os.getenv("PG_USER",     "postgres")
PG_PASSWORD = os.getenv("PG_PASSWORD", "password")
PG_DB       = os.getenv("PG_DB",       "banking")

st.set_page_config(
    page_title="Banking Simulator",
    page_icon="💳",
    layout="wide",
    initial_sidebar_state="expanded"
)

st.markdown("""
<style>
    @import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600&display=swap');

    :root {
        --bg:           #0a0b0d;
        --surface:      rgba(17, 19, 24, 0.75);
        --surface2:     rgba(25, 28, 36, 0.90);
        --border:       rgba(255, 255, 255, 0.05);
        --border-h:     rgba(99, 102, 241, 0.25);
        --text:         #f3f4f6;
        --muted:        #6b7280;
        --muted2:       #9ca3af;
        --accent:       #6366f1; /* Premium Indigo */
        --accent-dim:   #4f46e5;
        --accent-glow:  rgba(99, 102, 241, 0.15);
        --red:          #f87171;
        --green:        #34d399;
        --yellow:       #fbbf24;
        --blue:         #38bdf8;
        --purple:       #c084fc;
        color-scheme: dark;
    }

    /* ── Scrollbar ── */
    ::-webkit-scrollbar { width: 5px; height: 5px; }
    ::-webkit-scrollbar-track { background: transparent; }
    ::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.1); border-radius: 10px; }
    ::-webkit-scrollbar-thumb:hover { background: var(--accent); }

    /* ── Animations ── */
    @keyframes fadeIn {
        from { opacity: 0; transform: translateY(6px); }
        to   { opacity: 1; transform: translateY(0); }
    }
    @keyframes pulse-dot {
        0%, 100% { opacity: 1; filter: drop-shadow(0 0 2px currentColor); }
        50%       { opacity: 0.4; filter: drop-shadow(0 0 8px currentColor); }
    }

    html, body, [class*="css"] {
        font-family: 'Plus Jakarta Sans', sans-serif;
        color: var(--text);
        background-color: var(--bg);
        background-image:
            radial-gradient(circle at 50% -20%, rgba(99, 102, 241, 0.08) 0%, transparent 50%),
            radial-gradient(circle at 0% 40%, rgba(56, 189, 248, 0.02) 0%, transparent 40%);
        background-attachment: fixed;
    }

    .stApp, [data-testid="stAppViewContainer"] {
        background-color: transparent;
    }

    [data-testid="stHeader"] {
        background-color: rgba(10, 11, 13, 0.5);
        backdrop-filter: blur(16px);
        -webkit-backdrop-filter: blur(16px);
        border-bottom: 1px solid var(--border);
    }

    .block-container {
        padding-top: 2rem;
        padding-bottom: 2rem;
        max-width: 1320px;
    }

    /* ── Sidebar ── */
    [data-testid="stSidebar"] {
        background-color: #0d0f13 !important;
        border-right: 1px solid var(--border) !important;
    }
    [data-testid="stSidebar"] * { color: var(--text) !important; }

    /* ── Typography ── */
    h1, h2, h3 {
        font-family: 'Plus Jakarta Sans', sans-serif;
        font-weight: 700;
        letter-spacing: -0.02em;
    }

    .page-title {
        font-size: 2.2rem;
        font-weight: 800;
        letter-spacing: -0.03em;
        background: linear-gradient(135deg, #ffffff 30%, #a5b4fc 100%);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
        margin-bottom: 0.1rem;
    }

    .page-title-bar {
        width: 2.5rem;
        height: 4px;
        background: var(--accent);
        border-radius: 4px;
        margin-bottom: 0.8rem;
    }

    .page-subtitle {
        font-size: 0.95rem;
        color: var(--muted2);
        margin-bottom: 2rem;
        font-weight: 400;
    }

    /* ── Premium Cards Layout ── */
    .metric-card, .account-row, .transfer-card, .result-box {
        background: var(--surface);
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        border: 1px solid var(--border);
        border-radius: 12px;
        box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
        animation: fadeIn 0.4s cubic-bezier(0.16, 1, 0.3, 1) both;
        transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
    }
    
    .metric-card {
        padding: 1.25rem 1.5rem;
        margin-bottom: 1rem;
    }

    .account-row, .transfer-card {
        padding: 1.2rem 1.5rem;
        margin-bottom: 0.75rem;
    }

    .account-row:hover, .transfer-card:hover { 
        border-color: var(--border-h);
        transform: translateY(-2px);
        background: rgba(25, 28, 36, 0.5);
        box-shadow: 0 12px 24px -10px rgba(0, 0, 0, 0.4);
    }

    .metric-label {
        font-size: 0.75rem;
        color: var(--muted2);
        text-transform: uppercase;
        letter-spacing: 0.08em;
        font-weight: 600;
    }

    .metric-value {
        font-size: 1.75rem;
        font-weight: 700;
        color: #ffffff;
        font-family: 'JetBrains Mono', monospace;
        margin-top: 0.25rem;
        letter-spacing: -0.02em;
    }

    .metric-icon {
        font-size: 1.2rem;
        margin-bottom: 0.4rem;
        opacity: 0.9;
    }

    /* ── Avatars ── */
    .avatar {
        width: 38px;
        height: 38px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-weight: 700;
        font-size: 0.85rem;
        flex-shrink: 0;
        letter-spacing: -0.01em;
    }

    .account-name {
        font-weight: 600;
        color: #ffffff;
        font-size: 1rem;
    }

    .account-id {
        font-size: 0.75rem;
        color: var(--muted);
        font-family: 'JetBrains Mono', monospace;
        margin-top: 0.1rem;
    }

    .account-balance {
        font-family: 'JetBrains Mono', monospace;
        font-size: 1.1rem;
        font-weight: 600;
        color: #ffffff;
    }

    /* ── Transactions ── */
    .txn-row {
        background: rgba(255, 255, 255, 0.015);
        border: 1px solid rgba(255, 255, 255, 0.04);
        border-radius: 8px;
        padding: 0.9rem 1.2rem;
        margin-bottom: 0.5rem;
        transition: all 0.2s;
    }

    .txn-row:hover {
        background: rgba(255, 255, 255, 0.03);
        border-color: rgba(255, 255, 255, 0.08);
    }

    /* ── Status Badges ── */
    .status-badge {
        display: inline-flex;
        align-items: center;
        padding: 0.2rem 0.6rem;
        border-radius: 6px;
        font-size: 0.68rem;
        font-weight: 600;
        font-family: 'JetBrains Mono', monospace;
        letter-spacing: 0.02em;
        text-transform: uppercase;
    }

    .status-success   { background: rgba(52, 211, 153, 0.1); color: #34d399; border: 1px solid rgba(52, 211, 153, 0.15); }
    .status-pending   { background: rgba(251, 191, 36, 0.1);  color: #fbbf24; border: 1px solid rgba(251, 191, 36, 0.15); }
    .status-processing { background: rgba(56, 189, 248, 0.1); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.15); }
    .status-failed    { background: rgba(248, 113, 113, 0.1); color: #f87171; border: 1px solid rgba(248, 113, 113, 0.15); }

    /* ── Custom Section Headers ── */
    .section-header {
        font-size: 0.75rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        color: var(--muted2);
        margin-bottom: 1.2rem;
        margin-top: 1.8rem;
        display: flex;
        align-items: center;
        gap: 0.6rem;
    }
    
    .section-header::after {
        content: "";
        flex: 1;
        height: 1px;
        background: rgba(255, 255, 255, 0.05);
    }

    .divider {
        border: none;
        border-top: 1px solid var(--border);
        margin: 1.5rem 0;
    }

    .mono {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.82rem;
        color: var(--muted2);
    }

    /* ── Streamlit Form Native Component Overrides ── */
    .stButton > button {
        background: var(--accent) !important;
        color: #ffffff !important;
        border: none !important;
        border-radius: 8px !important;
        padding: 0.55rem 1.2rem !important;
        font-family: 'Plus Jakarta Sans', sans-serif !important;
        font-size: 0.9rem !important;
        font-weight: 600 !important;
        width: 100% !important;
        transition: all 0.2s ease !important;
        box-shadow: 0 4px 12px rgba(99, 102, 241, 0.2) !important;
    }

    .stButton > button:hover {
        background: var(--accent-dim) !important;
        transform: translateY(-1px) !important;
        box-shadow: 0 6px 16px rgba(99, 102, 241, 0.3) !important;
    }

    /* Form Fields Styling */
    .stSelectbox label, .stTextInput label, .stNumberInput label {
        font-size: 0.8rem !important;
        font-weight: 500 !important;
        color: var(--muted2) !important;
        letter-spacing: 0.02em !important;
        margin-bottom: 0.4rem !important;
    }

    .stTextInput input, .stNumberInput input, .stSelectbox > div > div {
        background-color: rgba(255, 255, 255, 0.02) !important;
        border: 1px solid rgba(255, 255, 255, 0.06) !important;
        border-radius: 8px !important;
        color: var(--text) !important;
        padding: 0.5rem 0.8rem !important;
    }

    .stTextInput input:focus, .stNumberInput input:focus, .stSelectbox > div > div:focus-within {
        border-color: var(--accent) !important;
        box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.15) !important;
        background-color: rgba(255, 255, 255, 0.04) !important;
    }

    /* Containers & Expanders */
    .result-box {
        padding: 1.25rem;
        margin-top: 1rem;
        border-left: 3px solid var(--accent);
        background: rgba(99, 102, 241, 0.02);
    }

    .result-key {
        font-size: 0.7rem;
        color: var(--muted2);
        text-transform: uppercase;
        letter-spacing: 0.05em;
        font-weight: 600;
    }

    .result-val {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.9rem;
        color: #ffffff;
        margin-top: 0.15rem;
        margin-bottom: 0.75rem;
    }

    .streamlit-expanderHeader {
        background: rgba(255, 255, 255, 0.01) !important;
        color: var(--text) !important;
        font-size: 0.88rem !important;
        font-weight: 500 !important;
        border-radius: 8px !important;
        border: 1px solid var(--border) !important;
    }
    
    .streamlit-expanderHeader:hover {
        background: rgba(255, 255, 255, 0.03) !important;
        border-color: var(--border-h) !important;
    }

    .streamlit-expanderContent {
        background: transparent !important;
        border: 1px solid var(--border) !important;
        border-top: none !important;
    }

    /* ── Sidebar Microservice Grid ── */
    .brand-block {
        display: flex;
        align-items: center;
        gap: 0.65rem;
        margin-bottom: 1.5rem;
        padding-bottom: 1.2rem;
        border-bottom: 1px solid var(--border);
    }

    .brand-icon {
        width: 34px;
        height: 34px;
        border-radius: 8px;
        background: linear-gradient(135deg, #6366f1, #4f46e5);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 1rem;
    }

    .sidebar-title {
        font-size: 0.95rem;
        font-weight: 700;
        color: #ffffff;
        letter-spacing: -0.01em;
    }

    .sidebar-sub {
        font-size: 0.65rem;
        color: var(--muted);
        letter-spacing: 0.05em;
        font-weight: 600;
    }

    .svc-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 0.5rem 0.65rem;
        border-radius: 6px;
        margin-bottom: 0.35rem;
        background: rgba(255, 255, 255, 0.01);
        border: 1px solid var(--border);
    }

    .svc-left {
        display: flex;
        align-items: center;
        gap: 0.5rem;
        font-size: 0.8rem;
    }

    .svc-port { font-size: 0.7rem; font-family: 'JetBrains Mono', monospace; color: var(--muted); }
    .svc-status { font-size: 0.68rem; font-weight: 600; text-transform: uppercase; }

    .service-dot {
        display: inline-block;
        width: 6px;
        height: 6px;
        border-radius: 50%;
    }
    .dot-green { background: var(--green); color: var(--green); animation: pulse-dot 2s infinite; }
    .dot-red { background: var(--red); color: var(--red); }

    .prog-bar-wrap { background: rgba(255, 255, 255, 0.04); border-radius: 4px; height: 4px; overflow: hidden; margin-top: 0.4rem; }
    .prog-bar-fill { height: 100%; border-radius: 4px; background: var(--accent); transition: width 0.5s ease; }

    #MainMenu, footer { visibility: hidden; }
</style>
""", unsafe_allow_html=True)


def get(url, path):
    try:
        r = requests.get(f"{url}{path}", timeout=5)
        return r.json() if r.status_code == 200 else None
    except Exception:
        return None


def post(url, path, payload):
    try:
        r = requests.post(f"{url}{path}", json=payload, timeout=10)
        return r.json(), r.status_code
    except Exception as e:
        return {"error": str(e)}, 500


def fmt_currency(amount):
    try:
        return f"₹ {float(amount):,.2f}"
    except Exception:
        return str(amount)


def fmt_id(uid):
    if uid and len(str(uid)) >= 8:
        return str(uid)[:8] + "..."
    return str(uid)


def fmt_date(dt_str):
    if not dt_str:
        return ""
    try:
        return datetime.fromisoformat(dt_str).strftime("%d %b %Y, %H:%M")
    except Exception:
        return dt_str


def status_badge(status):
    s = str(status).upper()
    cls = {
        "SUCCESS":    "status-success",
        "PENDING":    "status-pending",
        "PROCESSING": "status-processing",
        "FAILED":     "status-failed"
    }.get(s, "status-pending")
    return f'<span class="status-badge {cls}">{s}</span>'


with st.sidebar:
    st.markdown("""
    <div class="brand-block">
        <div class="brand-text">
            <div class="sidebar-title">NEXUS BANK</div>
            <div class="sidebar-sub">ENGINE ARCHITECTURE</div>
        </div>
    </div>
    """, unsafe_allow_html=True)

    sb_health  = get(SPRING_BOOT_URL, "/accounts") is not None
    gin_health = get(GIN_URL, "/health") is not None
    fa_health  = get(FASTAPI_URL, "/health") is not None

    st.markdown('<div class="section-header">Services</div>', unsafe_allow_html=True)

    def svc_row(name, port, healthy):
        dot    = "dot-green" if healthy else "dot-red"
        status = "active" if healthy else "offline"
        color  = var_color = "var(--green)" if healthy else "var(--muted)"
        st.markdown(
            f'<div class="svc-row">'
            f'<div class="svc-left"><span class="service-dot {dot}"></span>{name} <span class="svc-port">:{port}</span></div>'
            f'<span class="svc-status" style="color:{color};">{status}</span>'
            f'</div>',
            unsafe_allow_html=True
        )

    svc_row("Spring Boot", "8080", sb_health)
    svc_row("Gin",         "8081", gin_health)
    svc_row("FastAPI",     "8082", fa_health)

    st.markdown('<hr class="divider">', unsafe_allow_html=True)

    page = st.radio(
        "Navigate",
        ["Accounts", "Transfer", "Analytics", "Pipeline"],
        label_visibility="collapsed"
    )


if page == "Accounts":
    st.markdown('<div class="page-title-bar"></div>', unsafe_allow_html=True)
    st.markdown('<div class="page-title">Accounts</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">View accounts, balances and transaction history</div>', unsafe_allow_html=True)

    col1, col2 = st.columns([2, 1])

    with col1:
        st.markdown('<div class="section-header">All Accounts</div>', unsafe_allow_html=True)
        accounts = get(SPRING_BOOT_URL, "/accounts")

        if accounts:
            # Sophisticated muted professional color scheme for cards
            avatar_colors = [
                ("rgba(99, 102, 241, 0.15)", "#a5b4fc"), ("rgba(56, 189, 248, 0.15)", "#7dd3fc"),
                ("rgba(192, 132, 252, 0.15)", "#e9d5ff"), ("rgba(248, 113, 113, 0.15)", "#fca5a5"),
                ("rgba(251, 191, 36, 0.15)", "#fde047"), ("rgba(52, 211, 153, 0.15)", "#6ee7b7"),
            ]
            for idx, acc in enumerate(accounts):
                acc_id      = acc.get("id", "")
                acc_name    = acc.get("name", "Unknown")
                acc_balance = fmt_currency(acc.get("balance", 0))
                initials    = "".join(p[0].upper() for p in acc_name.split()[:2])
                av_bg, av_fg = avatar_colors[idx % len(avatar_colors)]

                st.markdown(f"""
                <div class="account-row">
                    <div style="display:flex;justify-content:space-between;align-items:center;gap:1rem;">
                        <div style="display:flex;align-items:center;gap:0.9rem;">
                            <div class="avatar" style="background:{av_bg};color:{av_fg};border:1px solid {av_fg}33;">{initials}</div>
                            <div>
                                <div class="account-name">{acc_name}</div>
                                <div class="account-id">{acc_id}</div>
                            </div>
                        </div>
                        <div class="account-balance">{acc_balance}</div>
                    </div>
                </div>
                """, unsafe_allow_html=True)

                with st.expander(f"View transactions for {acc_name}"):
                    txns = get(SPRING_BOOT_URL, f"/accounts/{acc_id}/transactions")
                    if txns:
                        for txn in txns:
                            is_debit     = txn.get("fromAccount") == acc_id
                            direction    = "Sent" if is_debit else "Received"
                            amount_color = "var(--red)" if is_debit else "var(--green)"
                            amount_sign  = "−" if is_debit else "+"
                            dir_arrow    = "→" if is_debit else "←"
                            dir_color    = "var(--red)" if is_debit else "var(--green)"
                            txn_id       = txn.get("id", "")
                            txn_mode     = txn.get('transferMode', '')
                            mode_colors  = {"IMPS": "#34d399", "NEFT": "#38bdf8", "RTGS": "#fbbf24"}
                            mode_color   = mode_colors.get(txn_mode.upper(), "#9ca3af")

                            st.markdown(f"""
                            <div class="txn-row">
                                <div style="display:flex;justify-content:space-between;align-items:center;">
                                    <div style="display:flex;align-items:center;gap:0.75rem;">
                                        <div style="width:24px;height:24px;border-radius:6px;background:rgba(255,255,255,0.02);
                                                    border:1px solid {dir_color}22;display:flex;align-items:center;
                                                    justify-content:center;font-size:0.75rem;color:{dir_color};flex-shrink:0;">{dir_arrow}</div>
                                        <div>
                                            <div style="font-size:0.85rem;font-weight:600;color:#f3f4f6;display:flex;align-items:center;gap:0.5rem;">
                                                {direction}
                                                <span style="font-size:0.6rem;font-weight:600;font-family:'JetBrains Mono',monospace;
                                                             color:{mode_color};background:rgba(255,255,255,0.03);
                                                             padding:0.1rem 0.4rem;border-radius:4px;border:1px solid {mode_color}11;">{txn_mode}</span>
                                            </div>
                                            <div style="font-size:0.72rem;color:var(--muted);font-family:'JetBrains Mono',monospace;margin-top:0.15rem;">Counterparty ID: {fmt_id(other_id if 'other_id' in locals() else (txn.get("toAccount") if is_debit else txn.get("fromAccount")))}</div>
                                            <div style="font-size:0.7rem;color:var(--muted);margin-top:0.1rem;">Timestamp: {fmt_date(txn.get('createdAt', ''))}</div>
                                        </div>
                                    </div>
                                    <div style="text-align:right;">
                                        <div style="font-family:'JetBrains Mono',monospace;font-size:0.9rem;font-weight:600;color:{amount_color};">
                                            {amount_sign} {fmt_currency(txn.get('amount', 0))}
                                        </div>
                                        <div style="margin-top:0.25rem;">{status_badge(txn.get('status', ''))}</div>
                                    </div>
                                </div>
                            </div>
                            """, unsafe_allow_html=True)

                            fraud_state_key = f"fraud_result_{acc_id}_{txn_id}"
                            col_fbtn, col_fsp = st.columns([1, 2])
                            with col_fbtn:
                                if st.button("Check Fraud Risk", key=f"fraud_btn_{acc_id}_{txn_id}", use_container_width=True):
                                    fraud_resp = get(FASTAPI_URL, f"/fraud/{txn_id}")
                                    st.session_state[fraud_state_key] = fraud_resp if fraud_resp and "risk_level" in fraud_resp else "error"

                            if fraud_state_key in st.session_state:
                                fd = st.session_state[fraud_state_key]
                                if fd == "error":
                                    st.warning("Could not assess fraud risk for this transaction.", icon=None)
                                else:
                                    rl = fd.get("risk_level", "LOW")
                                    r_colors  = {"HIGH": "var(--red)", "MEDIUM": "var(--yellow)", "LOW": "var(--green)"}
                                    r_bgs     = {"HIGH": "rgba(248,113,113,0.04)", "MEDIUM": "rgba(251,191,36,0.04)", "LOW": "rgba(52,211,153,0.04)"}
                                    r_borders = {"HIGH": "rgba(248,113,113,0.15)", "MEDIUM": "rgba(251,191,36,0.15)", "LOW": "rgba(52,211,153,0.15)"}
                                    rc  = r_colors.get(rl,  "#888")
                                    rb  = r_bgs.get(rl,    "rgba(255,255,255,0.02)")
                                    rbr = r_borders.get(rl, "rgba(255,255,255,0.05)")
                                    st.markdown(f"""
                                    <div style="background:{rb};border:1px solid {rbr};border-left:3px solid {rc};
                                                border-radius:6px;padding:0.6rem 1rem;margin-bottom:0.5rem;margin-top:-0.25rem;">
                                        <div style="display:flex;justify-content:space-between;align-items:center;">
                                            <div style="font-size:0.75rem;color:var(--muted2);">
                                                Fraud Risk:&nbsp;
                                                <span style="color:{rc};font-weight:700;font-family:'JetBrains Mono',monospace;">{rl} RISK</span>
                                            </div>
                                            <div style="font-size:0.72rem;color:var(--muted); font-style:italic;">{fd.get('reason', '')}</div>
                                        </div>
                                    </div>
                                    """, unsafe_allow_html=True)
                    else:
                        st.markdown('<div style="font-size:0.8rem;color:var(--muted);padding:0.5rem 0;">No transactions found.</div>', unsafe_allow_html=True)
        else:
            st.info("No accounts found or service unavailable.")

    with col2:
        st.markdown('<div class="section-header">Create Account</div>', unsafe_allow_html=True)

        with st.form("create_account"):
            name    = st.text_input("Full Name")
            tpin    = st.text_input("TPIN (4 digits)", max_chars=4, type="password", help="Set a 4-digit PIN for authorizing transfers")
            balance = st.number_input("Initial Balance", min_value=0.0, step=1000.0, value=50000.0)
            submitted = st.form_submit_button("Create Account")

        if submitted:
            if not name:
                st.error("Name is required.")
            elif not tpin or not tpin.isdigit() or len(tpin) != 4:
                st.error("TPIN must be exactly 4 digits.")
            else:
                result, code = post(SPRING_BOOT_URL, "/accounts", {
                    "name": name,
                    "initialBalance": balance,
                    "tpin": tpin
                })
                if code == 200:
                    st.success("Account initialized successfully.")
                    st.markdown(f"""
                    <div class="result-box">
                        <div class="result-key">Account ID</div>
                        <div class="result-val">{result.get('id', '')}</div>
                        <div class="result-key">Name</div>
                        <div class="result-val">{result.get('name', '')}</div>
                        <div class="result-key">Balance</div>
                        <div class="result-val">{fmt_currency(result.get('balance', 0))}</div>
                    </div>
                    """, unsafe_allow_html=True)
                else:
                    st.error("Failed to create account.")


elif page == "Transfer":
    st.markdown('<div class="page-title-bar"></div>', unsafe_allow_html=True)
    st.markdown('<div class="page-title">Transfer</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">Initiate NEFT, RTGS, or IMPS transfers</div>', unsafe_allow_html=True)

    accounts     = get(SPRING_BOOT_URL, "/accounts") or []
    account_map  = {acc["name"]: acc["id"] for acc in accounts}
    account_names = list(account_map.keys())

    col1, col2 = st.columns([1, 1])

    with col1:
        st.markdown('<div class="section-header">Initiate Transfer</div>', unsafe_allow_html=True)

        with st.form("transfer_form"):
            from_name = st.selectbox("From Account", account_names)
            to_name   = st.selectbox("To Account", account_names)
            amount    = st.number_input("Amount (Rs.)", min_value=1.0, step=1000.0, value=10000.0)
            mode      = st.selectbox("Transfer Mode", ["IMPS", "NEFT", "RTGS"])
            tpin      = st.text_input("TPIN", max_chars=4, type="password", help="Enter your 4-digit TPIN to authorize")
            submitted = st.form_submit_button("Initiate Transfer")

        if submitted:
            if from_name == to_name:
                st.error("Sender and receiver cannot be the same.")
            elif not tpin or not tpin.isdigit() or len(tpin) != 4:
                st.error("Enter a valid 4-digit TPIN.")
            elif not tpin or not tpin.isdigit() or len(tpin) != 4:
                st.error("Enter a valid 4-digit TPIN.")
            elif mode == "IMPS" and amount > 200000:
                st.error("IMPS maximum limit is Rs. 2,00,000.")
            elif mode == "RTGS" and amount < 200000:
                st.error("RTGS minimum amount is Rs. 2,00,000.")
            elif mode == "NEFT" and amount > 1000000:
                st.error("NEFT maximum limit is Rs. 10,00,000.")
            else:
                payload = {
                    "from_account":  account_map[from_name],
                    "to_account":    account_map[to_name],
                    "amount":        amount,
                    "transfer_mode": mode,
                    "tpin":          tpin
                }
                result, code = post(GIN_URL, "/process-transfer", payload)

                if code == 202:
                    transfer   = result.get("transfer", {})
                    txn_status = transfer.get("status", "")

                    if txn_status == "FAILED":
                        st.error("Transfer failed — incorrect TPIN or insufficient balance.")
                    else:
                        st.success(f"{mode} transfer initiated successfully.")

                    st.markdown(f"""
                    <div class="result-box">
                        <div class="result-key">Transfer ID</div>
                        <div class="result-val">{transfer.get('id', '')}</div>
                        <div class="result-key">Amount</div>
                        <div class="result-val">{fmt_currency(amount)}</div>
                        <div class="result-key">Mode</div>
                        <div class="result-val">{mode}</div>
                        <div class="result-key">Status</div>
                        <div class="result-val">{status_badge(txn_status)}</div>
                    </div>
                    """, unsafe_allow_html=True)

                    if txn_status != "FAILED":
                        if amount > 50000:
                            r_level, r_reason = "HIGH", f"Transaction exceeds ₹50,000 high-value threshold"
                            r_color, r_bg, r_border = "var(--red)", "rgba(248,113,113,0.04)", "rgba(248,113,113,0.15)"
                        elif amount > 10000:
                            r_level, r_reason = "MEDIUM", f"Moderate transaction of {fmt_currency(amount)}"
                            r_color, r_bg, r_border = "var(--yellow)", "rgba(251,191,36,0.04)", "rgba(251,191,36,0.15)"
                        else:
                            r_level, r_reason = "LOW", "Transaction within normal range"
                            r_color, r_bg, r_border = "var(--green)", "rgba(52,211,153,0.04)", "rgba(52,211,153,0.15)"

                        st.markdown(f"""
                        <div style="background:{r_bg};border:1px solid {r_border};
                                    border-left:4px solid {r_color};border-radius:8px;
                                    padding:1rem 1.25rem;margin-top:0.8rem;">
                            <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem;">
                                <div style="font-size:0.72rem;text-transform:uppercase;letter-spacing:0.05em;
                                            font-weight:700;color:var(--muted2);">Fraud Risk Assessment</div>
                                <span style="display:inline-block;padding:0.15rem 0.5rem;border-radius:4px;
                                             font-size:0.65rem;font-weight:700;font-family:'JetBrains Mono',monospace;
                                             background:{r_bg};color:{r_color};border:1px solid {r_border};">{r_level}</span>
                            </div>
                            <div style="font-size:0.85rem;color:var(--text);">{r_reason}</div>
                        </div>
                        """, unsafe_allow_html=True)

                    if txn_status not in ["FAILED", "SUCCESS"] and mode in ["NEFT", "RTGS"]:
                        delay = "30 seconds" if mode == "NEFT" else "15 seconds"

    with col2:
        st.markdown('<div class="section-header">Check Transfer Status</div>', unsafe_allow_html=True)

        transfer_id = st.text_input("Transfer ID")
        if st.button("Check Status"):
            if transfer_id:
                result = get(GIN_URL, f"/transfer/{transfer_id.strip()}")
                if result:
                    st.markdown(f"""
                    <div class="result-box">
                        <div class="result-key">Transfer ID</div>
                        <div class="result-val">{result.get('id', '')}</div>
                        <div class="result-key">Amount</div>
                        <div class="result-val">{fmt_currency(result.get('amount', 0))}</div>
                        <div class="result-key">Mode</div>
                        <div class="result-val">{result.get('transfer_mode', '')}</div>
                        <div class="result-key">Status</div>
                        <div class="result-val">{status_badge(result.get('status', ''))}</div>
                    </div>
                    """, unsafe_allow_html=True)
                else:
                    st.error("Transfer not found.")

    st.markdown('<hr class="divider">', unsafe_allow_html=True)
    st.markdown('<div class="section-header">All Transfers — Gin Processor</div>', unsafe_allow_html=True)

    all_transfers = get(GIN_URL, "/transfers")
    all_accounts  = get(SPRING_BOOT_URL, "/accounts") or []
    account_id_map = {acc["id"]: acc["name"] for acc in all_accounts}

    if all_transfers is not None:
        if len(all_transfers) == 0:
            st.markdown('<div style="font-size:0.8rem;color:var(--muted);padding:0.5rem 0;">No transfers found.</div>', unsafe_allow_html=True)
        else:
            total   = len(all_transfers)
            pending = sum(1 for t in all_transfers if t.get("status", "").upper() in ("PENDING", "PROCESSING"))
            failed  = sum(1 for t in all_transfers if t.get("status", "").upper() == "FAILED")
            success = sum(1 for t in all_transfers if t.get("status", "").upper() == "SUCCESS")

            m1, m2, m3, m4 = st.columns(4)
            for col, label, value, color in [
                (m1, "Total",      total,   "#ffffff"),
                (m2, "Success",    success, "var(--green)"),
                (m3, "In-Flight",  pending, "var(--yellow)"),
                (m4, "Failed",     failed,  "var(--red)"),
            ]:
                with col:
                    st.markdown(f"""
                    <div class="metric-card" style="padding:0.75rem 1.1rem;">
                        <div class="metric-label">{label}</div>
                        <div class="metric-value" style="font-size:1.2rem;color:{color};">{value}</div>
                    </div>
                    """, unsafe_allow_html=True)

            st.markdown("<div style='height:0.5rem'></div>", unsafe_allow_html=True)

            for t in all_transfers:
                from_id   = t.get("from_account", t.get("FromAccount", ""))
                to_id     = t.get("to_account",   t.get("ToAccount",   ""))
                from_name = account_id_map.get(from_id, fmt_id(from_id))
                to_name   = account_id_map.get(to_id,   fmt_id(to_id))
                mode      = t.get("transfer_mode", t.get("TransferMode", ""))
                amount    = t.get("amount",        t.get("Amount",       0))
                status    = t.get("status",        t.get("Status",       ""))
                tid       = t.get("id",            t.get("ID",           ""))
                created   = t.get("created_at",    t.get("CreatedAt",    ""))

                mode_colors = {"IMPS": "var(--green)", "NEFT": "var(--blue)", "RTGS": "var(--yellow)"}
                mode_color  = mode_colors.get(mode.upper(), "#888")

                st.markdown(f"""
                <div class="transfer-card">
                    <div style="display:flex;justify-content:space-between;align-items:center;">
                        <div style="flex:1;">
                            <div style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.35rem;">
                                <span style="font-size:0.65rem;font-weight:700;font-family:'JetBrains Mono',monospace;
                                             color:{mode_color};background:rgba(255,255,255,0.03);
                                             padding:0.1rem 0.45rem;border-radius:4px;border:1px solid rgba(255,255,255,0.05);">
                                    {mode}
                                </span>
                                {status_badge(status)}
                            </div>
                            <div style="font-size:0.88rem;color:#e5e7eb;font-weight:500;">
                                {from_name}
                                <span style="color:var(--muted);margin:0 0.3rem;">→</span>
                                {to_name}
                            </div>
                            <div style="font-size:0.72rem;color:var(--muted);font-family:'JetBrains Mono',monospace;margin-top:0.15rem;">
                                HASH ID: {tid}
                            </div>
                        </div>
                        <div style="text-align:right;">
                            <div style="font-family:'JetBrains Mono',monospace;font-size:0.95rem;font-weight:600;color:#ffffff;">
                                {fmt_currency(amount)}
                            </div>
                            <div style="font-size:0.7rem;color:var(--muted);margin-top:0.2rem;">
                                {fmt_date(created)}
                            </div>
                        </div>
                    </div>
                </div>
                """, unsafe_allow_html=True)
    else:
        st.warning("Could not reach Gin transfer processor.")


elif page == "Analytics":
    st.markdown('<div class="page-title-bar"></div>', unsafe_allow_html=True)
    st.markdown('<div class="page-title">Analytics</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">Transaction statistics and high-value transfers</div>', unsafe_allow_html=True)

    summary    = get(FASTAPI_URL, "/analytics/summary")
    high_value = get(FASTAPI_URL, "/analytics/high-value")

    if summary:
        col1, col2, col3 = st.columns(3)

        total_txns  = summary.get('total_transactions', 0)
        total_vol   = summary.get('total_volume', 0)
        hv_count    = high_value.get("count", 0) if high_value else 0

        with col1:
            st.markdown(f"""
            <div class="metric-card" style="border-top: 2px solid var(--blue);">
                <div class="metric-label">Total Transactions</div>
                <div class="metric-value">{total_txns}</div>
            </div>
            """, unsafe_allow_html=True)

        with col2:
            st.markdown(f"""
            <div class="metric-card" style="border-top: 2px solid var(--accent);">
                <div class="metric-label">Total Volume</div>
                <div class="metric-value">{fmt_currency(total_vol)}</div>
            </div>
            """, unsafe_allow_html=True)

        with col3:
            st.markdown(f"""
            <div class="metric-card" style="border-top: 2px solid var(--red);">
                <div class="metric-label">High Value Transfers (&gt;₹50k)</div>
                <div class="metric-value">{hv_count}</div>
            </div>
            """, unsafe_allow_html=True)

        col1, col2 = st.columns(2)

        with col1:
            st.markdown('<div class="section-header">By Transfer Mode</div>', unsafe_allow_html=True)
            mode_data = summary.get("by_transfer_mode", [])
            max_count = max((item.get('count', 0) for item in mode_data), default=1)
            mode_colors_map = {"IMPS": ("var(--green)", "var(--green)"), "NEFT": ("var(--blue)", "var(--blue)"), "RTGS": ("var(--yellow)", "var(--yellow)")}
            for item in mode_data:
                m = item.get('mode', '')
                c = item.get('count', 0)
                pct = int((c / max_count) * 100) if max_count else 0
                fs, fe = mode_colors_map.get(m.upper(), ("#9ca3af", "#6b7280"))
                st.markdown(f"""
                <div class="account-row" style="padding:0.9rem 1.3rem;">
                    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem;">
                        <span style="font-weight:600;font-size:0.85rem;color:{fs};">{m}</span>
                        <span style="font-family:'JetBrains Mono',monospace;font-size:0.8rem;color:var(--muted2);">{c} transactions</span>
                    </div>
                    <div class="prog-bar-wrap">
                        <div class="prog-bar-fill" style="width:{pct}%;background:{fs};"></div>
                    </div>
                </div>
                """, unsafe_allow_html=True)

        with col2:
            st.markdown('<div class="section-header">By Status</div>', unsafe_allow_html=True)
            status_data = summary.get("by_status", [])
            max_s = max((item.get('count', 0) for item in status_data), default=1)
            status_colors_map = {"SUCCESS": ("var(--green)", "var(--green)"), "PENDING": ("var(--yellow)", "var(--yellow)"),
                                  "PROCESSING": ("var(--blue)", "var(--blue)"), "FAILED": ("var(--red)", "var(--red)")}
            for item in status_data:
                s = item.get('status', '')
                c = item.get('count', 0)
                pct = int((c / max_s) * 100) if max_s else 0
                fs, fe = status_colors_map.get(s.upper(), ("#9ca3af", "#6b7280"))
                st.markdown(f"""
                <div class="account-row" style="padding:0.9rem 1.3rem;">
                    <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:0.4rem;">
                        <span>{status_badge(s)}</span>
                        <span style="font-family:'JetBrains Mono',monospace;font-size:0.8rem;color:var(--muted2);">{c}</span>
                    </div>
                    <div class="prog-bar-wrap">
                        <div class="prog-bar-fill" style="width:{pct}%;background:{fs};"></div>
                    </div>
                </div>
                """, unsafe_allow_html=True)

    st.markdown('<div class="section-header">High Value Transfers</div>', unsafe_allow_html=True)

    if high_value and high_value.get("transactions"):
        for txn in high_value["transactions"]:
            st.markdown(f"""
            <div class="transfer-card">
                <div style="display:flex;justify-content:space-between;align-items:center;">
                    <div>
                        <div class="mono" style="color:#ffffff; font-weight:500;">{txn.get('id', '')}</div>
                        <div style="font-size:0.72rem;color:var(--muted);margin-top:0.2rem;">{fmt_date(txn.get('createdAt', ''))}</div>
                    </div>
                    <div style="text-align:right;">
                        <div class="account-balance" style="color:var(--text);">{fmt_currency(txn.get('amount', 0))}</div>
                        <div style="margin-top:0.25rem;">{status_badge(txn.get('status', ''))}</div>
                    </div>
                </div>
            </div>
            """, unsafe_allow_html=True)
    else:
        st.info("No high value transfers found.")


# ─────────────────────────────────────────────────────────────────────────────
# PIPELINE PAGE  — reads from transfer_pipeline table (written by Airflow ETL)
# ─────────────────────────────────────────────────────────────────────────────
elif page == "Pipeline":
    st.markdown('<div class="page-title-bar"></div>', unsafe_allow_html=True)
    st.markdown('<div class="page-title">Transfer Pipeline</div>', unsafe_allow_html=True)
    st.markdown(
        '<div class="page-subtitle">'
        'Live view synced every 5 min by Airflow — '
        'shows MongoDB transfers vs PostgreSQL settlement. '
        'In-flight = initiated but not yet settled.'
        '</div>',
        unsafe_allow_html=True
    )

    # Helper to query Postgres directly
    def query_pipeline():
        if psycopg2 is None:
            return None, "psycopg2 not installed"
        try:
            conn = psycopg2.connect(
                host=PG_HOST, user=PG_USER,
                password=PG_PASSWORD, database=PG_DB,
                connect_timeout=5,
            )
            cur = conn.cursor()
            cur.execute("""
                SELECT
                    mongo_transfer_id,
                    from_account_id,
                    to_account_id,
                    amount,
                    transfer_mode,
                    mongo_status,
                    pg_status,
                    is_in_flight,
                    initiated_at,
                    settled_at,
                    settlement_lag_secs,
                    etl_synced_at
                FROM transfer_pipeline
                ORDER BY initiated_at DESC
                LIMIT 100;
            """)
            rows = cur.fetchall()
            cur.close()
            conn.close()
            return rows, None
        except Exception as e:
            return None, str(e)

    rows, err = query_pipeline()

    if err:
        if "transfer_pipeline" in (err or ""):
            st.warning("The pipeline table doesn't exist yet. Trigger the Airflow DAG first at http://localhost:8083")
        else:
            st.error(f"Could not connect to database: {err}")
    elif rows is None:
        st.warning("No data yet — trigger the Airflow ETL DAG at http://localhost:8083")
    else:
        # ── Summary metrics ─────────────────────────────────────────────
        total      = len(rows)
        in_flight  = sum(1 for r in rows if r[7])   # is_in_flight
        settled    = total - in_flight
        avg_lag    = (
            round(sum(r[10] for r in rows if r[10]) /
                  max(1, sum(1 for r in rows if r[10])), 1)
            if rows else 0
        )

        c1, c2, c3, c4 = st.columns(4)
        for col, label, val, color in [
            (c1, "Total Tracked",    total,     "#ffffff"),
            (c2, "In-Flight",        in_flight, "var(--yellow)"),
            (c3, "Settled",          settled,   "var(--green)"),
            (c4, "Avg Lag (s)",      avg_lag,   "var(--blue)"),
        ]:
            with col:
                st.markdown(f"""
                <div class="metric-card" style="padding:0.75rem 1.1rem;">
                    <div class="metric-label">{label}</div>
                    <div class="metric-value" style="font-size:1.2rem;color:{color};">{val}</div>
                </div>
                """, unsafe_allow_html=True)

        # ── In-flight banner ────────────────────────────────────────────
        if in_flight > 0:
            st.markdown(f"""
            <div style="background:rgba(251,191,36,0.06);border:1px solid rgba(251,191,36,0.2);
                        border-left:4px solid var(--yellow);border-radius:8px;
                        padding:0.8rem 1.2rem;margin:1rem 0;">
                <span style="font-weight:700;color:var(--yellow);">{in_flight} transfer(s) currently in-flight</span>
                <span style="color:var(--muted);font-size:0.85rem;"> — in MongoDB, not yet in Postgres</span>
            </div>
            """, unsafe_allow_html=True)

        # ── Row-by-row table ────────────────────────────────────────────
        st.markdown('<div class="section-header">All Transfers</div>', unsafe_allow_html=True)

        mode_colors = {"IMPS": "#34d399", "NEFT": "#38bdf8", "RTGS": "#fbbf24"}

        for r in rows:
            (
                mongo_id, from_acc, to_acc, amount, mode,
                m_status, pg_status, is_inflight,
                initiated, settled_at, lag, synced_at
            ) = r

            mode_color   = mode_colors.get(str(mode).upper(), "#9ca3af")
            badge_color  = "var(--yellow)" if is_inflight else "var(--green)"
            badge_label  = "IN-FLIGHT" if is_inflight else "SETTLED"
            badge_bg     = "rgba(251,191,36,0.08)" if is_inflight else "rgba(52,211,153,0.08)"
            badge_border = "rgba(251,191,36,0.2)"  if is_inflight else "rgba(52,211,153,0.2)"

            lag_str = f"{lag}s" if lag is not None else "—"
            pg_str  = pg_status if pg_status else "not settled yet"
            pg_color = "var(--muted)" if not pg_status else "var(--green)" if pg_status == "SUCCESS" else "var(--red)"

            st.markdown(f"""
            <div class="transfer-card" style="margin-bottom:0.6rem;">
                <div style="display:flex;justify-content:space-between;align-items:center;">
                    <div style="display:flex;align-items:center;gap:0.75rem;">
                        <span style="font-size:0.65rem;font-weight:700;font-family:'JetBrains Mono',monospace;
                                     color:{mode_color};background:rgba(255,255,255,0.03);
                                     padding:0.2rem 0.5rem;border-radius:4px;border:1px solid {mode_color}22;"
                        >{mode}</span>
                        <div>
                            <div class="mono" style="color:#fff;font-weight:500;font-size:0.8rem;">
                                {fmt_id(mongo_id)}
                            </div>
                            <div style="font-size:0.7rem;color:var(--muted);margin-top:0.1rem;">
                                {fmt_date(initiated.isoformat() if initiated else '')}
                            </div>
                        </div>
                    </div>
                    <div style="display:flex;align-items:center;gap:1rem;">
                        <div style="text-align:right;">
                            <div style="font-size:0.72rem;color:var(--muted2);">
                                MongoDB: <span style="color:var(--text);font-weight:600;">{m_status}</span>
                            </div>
                            <div style="font-size:0.72rem;color:var(--muted2);margin-top:0.15rem;">
                                Postgres: <span style="color:{pg_color};font-weight:600;">{pg_str}</span>
                            </div>
                            <div style="font-size:0.7rem;color:var(--muted);margin-top:0.15rem;">
                                Lag: {lag_str}
                            </div>
                        </div>
                        <span style="display:inline-block;padding:0.25rem 0.6rem;border-radius:6px;
                                     font-size:0.65rem;font-weight:700;font-family:'JetBrains Mono',monospace;
                                     color:{badge_color};background:{badge_bg};border:1px solid {badge_border};"
                        >{badge_label}</span>
                    </div>
                </div>
                <div style="font-size:0.7rem;color:var(--muted);margin-top:0.4rem;">
                    {fmt_id(from_acc)} → {fmt_id(to_acc)}
                    &nbsp;·&nbsp; {fmt_currency(amount)}
                    &nbsp;·&nbsp; ETL synced: {fmt_date(synced_at.isoformat() if synced_at else '')}
                </div>
            </div>
            """, unsafe_allow_html=True)

        st.markdown(
            f'<div style="font-size:0.72rem;color:var(--muted);text-align:right;margin-top:0.5rem;">'
            f'Airflow DAG runs every 5 min · '
            f'<a href="http://localhost:8083" target="_blank" style="color:var(--accent);">Open Airflow UI</a>'
            f'</div>',
            unsafe_allow_html=True
        )