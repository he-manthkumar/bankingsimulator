import streamlit as st
import requests
import os
from datetime import datetime

SPRING_BOOT_URL = os.getenv("SPRING_BOOT_URL", "http://localhost:8080")
GIN_URL = os.getenv("GIN_URL", "http://localhost:8081")
FASTAPI_URL = os.getenv("FASTAPI_URL", "http://localhost:8082")

st.set_page_config(
    page_title="Banking Simulator",
    page_icon=None,
    layout="wide",
    initial_sidebar_state="expanded"
)

st.markdown("""
<style>
    @import url('https://fonts.googleapis.com/css2?family=Syne:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500&family=DM+Sans:wght@300;400;500&display=swap');

    :root {
        --bg:        #0a0a0a;
        --surface:   rgba(24, 24, 24, 0.65);
        --border:    rgba(255, 255, 255, 0.08);
        --text:      #f5f5f5;
        --muted:     #9ca3af;
        --accent:    #d4f379;
        --accent-dim:#bce545;
        --red:       #fc8181;
        --green:     #d4f379;
        --yellow:    #fbd38d;
        --blue:      #90cdf4;
        color-scheme: dark;
    }

    html, body, [class*="css"] {
        font-family: 'DM Sans', sans-serif;
        color: var(--text);
        background-color: var(--bg);
        background-image: radial-gradient(circle at 50% 0%, rgba(212, 243, 121, 0.04) 0%, transparent 60%);
        background-attachment: fixed;
    }

    .stApp, [data-testid="stAppViewContainer"] {
        background-color: transparent;
    }

    [data-testid="stHeader"] {
        background-color: rgba(10, 10, 10, 0.8);
        backdrop-filter: blur(12px);
        -webkit-backdrop-filter: blur(12px);
        border-bottom: 1px solid var(--border);
    }

    .block-container {
        padding-top: 2rem;
        padding-bottom: 2rem;
    }

    [data-testid="stSidebar"],
    [data-testid="stSidebar"] > div {
        background-color: rgba(15, 15, 15, 0.95) !important;
        border-right: 1px solid var(--border) !important;
        backdrop-filter: blur(10px) !important;
    }

    [data-testid="stSidebar"] * { color: var(--text) !important; }

    [data-testid="stSidebar"] .stRadio label {
        font-size: 0.88rem;
        font-weight: 500;
        transition: color 0.2s;
    }

    h1, h2, h3 {
        font-family: 'Syne', sans-serif;
        font-weight: 700;
        letter-spacing: -0.03em;
        color: var(--text);
    }

    .page-title {
        font-family: 'Syne', sans-serif;
        font-size: 2.2rem;
        font-weight: 800;
        color: var(--text);
        margin-bottom: 0.15rem;
        letter-spacing: -0.04em;
        background: linear-gradient(90deg, #fff, #a1a1aa);
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .page-subtitle {
        font-size: 0.95rem;
        color: var(--muted);
        margin-bottom: 2rem;
        font-weight: 400;
    }

    .metric-card, .account-row, .transfer-card, .result-box {
        background: var(--surface);
        backdrop-filter: blur(16px);
        -webkit-backdrop-filter: blur(16px);
        border: 1px solid var(--border);
        border-radius: 12px;
        transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
        box-shadow: 0 4px 20px -2px rgba(0, 0, 0, 0.2);
    }
    
    .metric-card {
        padding: 1.5rem;
        margin-bottom: 1rem;
    }
    
    .account-row, .transfer-card {
        padding: 1rem 1.5rem;
        margin-bottom: 0.8rem;
    }

    .account-row:hover, .transfer-card:hover { 
        border-color: rgba(212, 243, 121, 0.25);
        transform: translateY(-2px);
        box-shadow: 0 8px 24px -4px rgba(0, 0, 0, 0.4);
    }

    .metric-label {
        font-size: 0.75rem;
        color: var(--muted);
        text-transform: uppercase;
        letter-spacing: 0.12em;
        font-weight: 600;
    }

    .metric-value {
        font-size: 1.8rem;
        font-weight: 700;
        color: var(--text);
        font-family: 'JetBrains Mono', monospace;
        margin-top: 0.4rem;
        text-shadow: 0 2px 10px rgba(255,255,255,0.05);
    }

    .account-name {
        font-family: 'Syne', sans-serif;
        font-weight: 600;
        color: var(--text);
        font-size: 1rem;
    }

    .account-id {
        font-size: 0.75rem;
        color: var(--muted);
        font-family: 'JetBrains Mono', monospace;
        margin-top: 0.2rem;
    }

    .account-balance {
        font-family: 'JetBrains Mono', monospace;
        font-size: 1.1rem;
        font-weight: 600;
        color: var(--accent);
    }

    .txn-row {
        background: rgba(30, 30, 30, 0.4);
        border: 1px solid rgba(255, 255, 255, 0.05);
        border-radius: 8px;
        padding: 0.8rem 1.2rem;
        margin-bottom: 0.5rem;
        transition: background 0.2s;
    }
    
    .txn-row:hover {
        background: rgba(40, 40, 40, 0.5);
    }

    .status-badge {
        display: inline-block;
        padding: 0.25rem 0.8rem;
        border-radius: 6px;
        font-size: 0.7rem;
        font-weight: 700;
        font-family: 'JetBrains Mono', monospace;
        letter-spacing: 0.08em;
        text-transform: uppercase;
        box-shadow: 0 2px 8px rgba(0,0,0,0.1);
    }

    .status-success      { background: rgba(212,243,121,0.15);  color: var(--green);  border: 1px solid rgba(212,243,121,0.25); }
    .status-pending      { background: rgba(251,211,141,0.15);  color: var(--yellow); border: 1px solid rgba(251,211,141,0.25); }
    .status-processing   { background: rgba(144,205,244,0.15); color: var(--blue);  border: 1px solid rgba(144,205,244,0.25); }
    .status-failed       { background: rgba(252,129,129,0.15);   color: var(--red);    border: 1px solid rgba(252,129,129,0.25); }
    .status-compensated  { background: rgba(246,173,85,0.15);   color: #f6ad55;       border: 1px solid rgba(246,173,85,0.25); }
    .status-cancelled    { background: rgba(113,128,150,0.15);  color: #a0aec0;       border: 1px solid rgba(113,128,150,0.25); }
    .status-waiting      { background: rgba(144,205,244,0.15);  color: var(--blue);  border: 1px solid rgba(144,205,244,0.25); }

    .section-header {
        font-family: 'Syne', sans-serif;
        font-size: 0.75rem;
        font-weight: 700;
        text-transform: uppercase;
        letter-spacing: 0.15em;
        color: var(--muted);
        margin-bottom: 1rem;
        margin-top: 1.8rem;
        display: flex;
        align-items: center;
    }
    
    .section-header::after {
        content: "";
        flex: 1;
        height: 1px;
        background: linear-gradient(90deg, var(--border), transparent);
        margin-left: 1rem;
    }

    .divider {
        border: none;
        border-top: 1px solid var(--border);
        margin: 2rem 0;
    }

    .mono {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.85rem;
        color: var(--muted);
    }

    .stButton > button {
        background: linear-gradient(135deg, var(--accent), var(--accent-dim)) !important;
        color: #050505 !important;
        border: none !important;
        border-radius: 8px !important;
        padding: 0.65rem 1.5rem !important;
        font-family: 'Syne', sans-serif !important;
        font-size: 0.95rem !important;
        font-weight: 700 !important;
        width: 100% !important;
        cursor: pointer !important;
        letter-spacing: 0.02em !important;
        transition: all 0.2s ease !important;
        box-shadow: 0 4px 15px rgba(212, 243, 121, 0.15) !important;
    }

    .stButton > button:hover {
        transform: translateY(-2px) !important;
        box-shadow: 0 6px 20px rgba(212, 243, 121, 0.25) !important;
        filter: brightness(1.05) !important;
    }
    
    .stButton > button:active {
        transform: translateY(0) !important;
    }

    .stSelectbox label,
    .stTextInput label,
    .stNumberInput label {
        font-size: 0.85rem !important;
        font-weight: 600 !important;
        color: #999 !important;
        letter-spacing: 0.05em !important;
        text-transform: uppercase !important;
        margin-bottom: 0.4rem !important;
    }

    .stTextInput input,
    .stNumberInput input {
        background-color: rgba(255, 255, 255, 0.03) !important;
        border: 1px solid var(--border) !important;
        border-radius: 8px !important;
        color: var(--text) !important;
        font-family: 'DM Sans', sans-serif !important;
        padding: 0.6rem 1rem !important;
        transition: all 0.2s ease !important;
    }

    .stTextInput input:focus,
    .stNumberInput input:focus {
        background-color: rgba(255, 255, 255, 0.05) !important;
        border-color: var(--accent) !important;
        box-shadow: 0 0 0 3px rgba(212, 243, 121, 0.15) !important;
    }

    .stSelectbox > div > div {
        background-color: rgba(255, 255, 255, 0.03) !important;
        border: 1px solid var(--border) !important;
        color: var(--text) !important;
        border-radius: 8px !important;
        transition: all 0.2s ease !important;
    }
    
    .stSelectbox > div > div:focus-within {
        border-color: var(--accent) !important;
        box-shadow: 0 0 0 3px rgba(212, 243, 121, 0.15) !important;
    }

    [data-testid="stAlert"] {
        border-radius: 10px !important;
        border: 1px solid var(--border) !important;
        font-family: 'DM Sans', sans-serif !important;
        font-size: 0.95rem !important;
        font-weight: 500 !important;
        backdrop-filter: blur(12px) !important;
        -webkit-backdrop-filter: blur(12px) !important;
    }

    /* Success alert */
    [data-testid="stAlert"][data-baseweb="notification"] {
        background: rgba(212,243,121,0.08) !important;
        border-left: 4px solid var(--green) !important;
        color: var(--green) !important;
    }

    div[data-testid="stAlert"] p,
    div[data-testid="stAlert"] div {
        color: inherit !important;
    }

    /* Streamlit success/error/info overrides */
    .element-container .stAlert > div {
        color: var(--text) !important;
    }

    /* Success */
    [class*="AlertSuccess"],
    [data-baseweb="notification"][kind="positive"] {
        background-color: rgba(212,243,121,0.08) !important;
        border-left: 4px solid var(--green) !important;
        color: var(--green) !important;
    }

    [class*="AlertError"],
    [data-baseweb="notification"][kind="negative"] {
        background-color: rgba(252,129,129,0.08) !important;
        border-left: 4px solid var(--red) !important;
        color: var(--red) !important;
    }

    [class*="AlertInfo"],
    [data-baseweb="notification"][kind="info"] {
        background-color: rgba(144,205,244,0.08) !important;
        border-left: 4px solid var(--blue) !important;
        color: var(--blue) !important;
    }

    [class*="AlertWarning"],
    [data-baseweb="notification"][kind="warning"] {
        background-color: rgba(251,211,141,0.08) !important;
        border-left: 4px solid var(--yellow) !important;
        color: var(--yellow) !important;
    }

    div[role="alert"] {
        color: var(--text) !important;
    }
    div[role="alert"] p {
        color: inherit !important;
        font-size: 0.95rem !important;
        font-weight: 500 !important;
    }

    .sidebar-title {
        font-family: 'Syne', sans-serif;
        font-size: 1.25rem;
        font-weight: 800;
        color: var(--text);
        letter-spacing: -0.02em;
        margin-bottom: 0.3rem;
        background: linear-gradient(90deg, #fff, var(--accent));
        -webkit-background-clip: text;
        -webkit-text-fill-color: transparent;
    }

    .sidebar-sub {
        font-size: 0.8rem;
        color: var(--muted);
        margin-bottom: 1.8rem;
    }

    .service-dot {
        display: inline-block;
        width: 8px;
        height: 8px;
        border-radius: 50%;
        margin-right: 8px;
    }

    .dot-green { background: var(--green); box-shadow: 0 0 8px var(--green); }
    .dot-red   { background: var(--red);   box-shadow: 0 0 8px var(--red); }
    
    .result-box {
        padding: 1.5rem;
        margin-top: 1.2rem;
        border-left: 4px solid var(--accent);
    }

    .result-key {
        font-size: 0.75rem;
        color: var(--muted);
        text-transform: uppercase;
        letter-spacing: 0.12em;
        font-weight: 600;
    }

    .result-val {
        font-family: 'JetBrains Mono', monospace;
        font-size: 0.95rem;
        color: var(--text);
        margin-top: 0.2rem;
        margin-bottom: 1rem;
        font-weight: 500;
    }

    .streamlit-expanderHeader {
        background: rgba(255, 255, 255, 0.02) !important;
        color: var(--accent) !important;
        font-size: 0.9rem !important;
        font-weight: 600 !important;
        border-radius: 8px !important;
        border: 1px solid var(--border) !important;
        transition: background 0.2s !important;
    }
    
    .streamlit-expanderHeader:hover {
        background: rgba(255, 255, 255, 0.05) !important;
    }

    .streamlit-expanderContent {
        background: rgba(15, 15, 15, 0.5) !important;
        border: 1px solid var(--border) !important;
        border-top: none !important;
        border-bottom-left-radius: 8px !important;
        border-bottom-right-radius: 8px !important;
    }

    .stRadio > div {
        gap: 0.4rem !important;
    }

    .stRadio [data-testid="stMarkdownContainer"] p {
        font-size: 0.95rem !important;
        font-weight: 500 !important;
    }

    #MainMenu { visibility: hidden; }
    footer { visibility: hidden; }
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
        return f"Rs. {float(amount):,.2f}"
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
        "SUCCESS":     "status-success",
        "PENDING":     "status-pending",
        "PROCESSING":  "status-processing",
        "FAILED":      "status-failed",
        "COMPENSATED": "status-compensated",
        "CANCELLED":   "status-cancelled",
        "WAITING":     "status-waiting",
        "DEBITING":    "status-processing",
        "CREDITING":   "status-processing",
        "COMPLETED":   "status-success",
    }.get(s, "status-pending")
    return f'<span class="status-badge {cls}">{s}</span>'


with st.sidebar:
    st.markdown('<div class="sidebar-title">Banking Simulator</div>', unsafe_allow_html=True)

    sb_health  = get(SPRING_BOOT_URL, "/accounts") is not None
    gin_health = get(GIN_URL, "/health") is not None
    fa_health  = get(FASTAPI_URL, "/health") is not None

    st.markdown('<div class="section-header">Services</div>', unsafe_allow_html=True)

    def svc_row(name, port, healthy):
        dot    = "dot-green" if healthy else "dot-red"
        status = "online" if healthy else "offline"
        color  = "#c8f060" if healthy else "#666"
        st.markdown(
            f'<div style="font-size:0.85rem;margin-bottom:0.5rem;color:#ccc;">'
            f'<span class="service-dot {dot}"></span>{name} '
            f'<span style="color:#555;font-size:0.78rem;">:{port}</span>'
            f'<span style="float:right;font-size:0.75rem;color:{color};">{status}</span>'
            f'</div>',
            unsafe_allow_html=True
        )

    svc_row("Spring Boot", "8080", sb_health)
    svc_row("Gin",         "8081", gin_health)
    svc_row("FastAPI",     "8082", fa_health)

    st.markdown('<hr class="divider">', unsafe_allow_html=True)

    page = st.radio(
        "Navigate",
        ["Accounts", "Transfer", "Analytics"],
        label_visibility="collapsed"
    )


if page == "Accounts":
    st.markdown('<div class="page-title">Accounts</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">View accounts, balances, and transaction history</div>', unsafe_allow_html=True)

    col1, col2 = st.columns([2, 1])

    with col1:
        st.markdown('<div class="section-header">All Accounts</div>', unsafe_allow_html=True)
        accounts = get(SPRING_BOOT_URL, "/accounts")

        if accounts:
            for acc in accounts:
                acc_id      = acc.get("id", "")
                acc_name    = acc.get("name", "Unknown")
                acc_balance = fmt_currency(acc.get("balance", 0))

                st.markdown(f"""
                <div class="account-row">
                    <div style="display:flex;justify-content:space-between;align-items:center;">
                        <div>
                            <div class="account-name">{acc_name}</div>
                            <div class="account-id">{acc_id}</div>
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
                            other_id     = txn.get("toAccount") if is_debit else txn.get("fromAccount")
                            amount_color = "#ff4d4d" if is_debit else "#c8f060"
                            amount_sign  = "−" if is_debit else "+"

                            st.markdown(f"""
                            <div class="txn-row">
                                <div style="display:flex;justify-content:space-between;align-items:center;">
                                    <div>
                                        <div style="font-size:0.85rem;font-weight:500;color:#f0f0f0;">
                                            {direction} &middot; {txn.get('transferMode', '')}
                                        </div>
                                        <div style="font-size:0.7rem;color:#555;font-family:'JetBrains Mono',monospace;margin-top:0.15rem;">
                                            {fmt_id(other_id)}
                                        </div>
                                        <div style="font-size:0.7rem;color:#444;margin-top:0.1rem;">
                                            {fmt_date(txn.get('createdAt', ''))}
                                        </div>
                                    </div>
                                    <div style="text-align:right;">
                                        <div style="font-family:'JetBrains Mono',monospace;font-size:0.95rem;font-weight:500;color:{amount_color};">
                                            {amount_sign} {fmt_currency(txn.get('amount', 0))}
                                        </div>
                                        <div style="margin-top:0.35rem;">{status_badge(txn.get('status', ''))}</div>
                                    </div>
                                </div>
                            </div>
                            """, unsafe_allow_html=True)
                    else:
                        st.markdown('<div style="font-size:0.85rem;color:#555;padding:0.5rem 0;">No transactions found.</div>', unsafe_allow_html=True)
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
                    st.success("Account created successfully.")
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
    st.markdown('<div class="page-title">Transfer</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">Initiate NEFT, RTGS, or IMPS transfers</div>', unsafe_allow_html=True)

    # ── Session-state initialisation ────────────────────────────────────────
    if "status_result" not in st.session_state:
        st.session_state.status_result = None
    if "status_transfer_id" not in st.session_state:
        st.session_state.status_transfer_id = ""
    if "cancel_result" not in st.session_state:
        st.session_state.cancel_result = None   # None | "success" | "conflict" | "error"
    if "cancel_message" not in st.session_state:
        st.session_state.cancel_message = ""

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

                    if txn_status not in ["FAILED", "SUCCESS"] and mode in ["NEFT", "RTGS"]:
                        delay = "30 seconds" if mode == "NEFT" else "15 seconds"

    with col2:
        st.markdown('<div class="section-header">Check Transfer Status</div>', unsafe_allow_html=True)

        # ── Input + Check Status button ──────────────────────────────────────
        transfer_id_input = st.text_input(
            "Transfer ID",
            value=st.session_state.status_transfer_id,
            key="transfer_id_input"
        )

        if st.button("Check Status"):
            # Persist the ID and fetch fresh status; clear any stale cancel result
            st.session_state.status_transfer_id = transfer_id_input.strip()
            st.session_state.cancel_result  = None
            st.session_state.cancel_message = ""
            st.session_state.status_result  = get(GIN_URL, f"/transfer/{transfer_id_input.strip()}/status")

        # ── Cancel button — top-level, outside the Check Status block ────────
        # Only shown when the last fetched state is WAITING
        wf_state_cached = (st.session_state.status_result or {}).get("state", "")
        if st.session_state.status_transfer_id and wf_state_cached == "WAITING":
            st.markdown("<div style='height:0.4rem'></div>", unsafe_allow_html=True)
            if st.button("🚫 Cancel Transfer", key="cancel_btn"):
                tid = st.session_state.status_transfer_id
                try:
                    cancel_resp = requests.put(
                        f"{GIN_URL}/transfer/{tid}/cancel",
                        timeout=10
                    )
                    if cancel_resp.status_code == 200:
                        st.session_state.cancel_result  = "success"
                        st.session_state.cancel_message = "✅ Cancellation confirmed — transfer will not be settled"
                        # Refresh status so WAITING badge updates
                        st.session_state.status_result = get(GIN_URL, f"/transfer/{tid}/status")
                    elif cancel_resp.status_code == 409:
                        st.session_state.cancel_result  = "conflict"
                        st.session_state.cancel_message = cancel_resp.json().get("error", "Could not cancel")
                    else:
                        st.session_state.cancel_result  = "error"
                        st.session_state.cancel_message = "Could not reach workflow engine"
                except Exception as e:
                    st.session_state.cancel_result  = "error"
                    st.session_state.cancel_message = f"Request failed: {e}"

        # ── Render cancel feedback ────────────────────────────────────────────
        if st.session_state.cancel_result == "success":
            st.success(st.session_state.cancel_message)
        elif st.session_state.cancel_result in ("conflict", "error"):
            st.error(st.session_state.cancel_message)

        # ── Render status result ──────────────────────────────────────────────
        result = st.session_state.status_result
        if result is not None:
            wf_state   = result.get("state", "")
            elapsed    = result.get("elapsed_seconds", 0)
            remaining  = result.get("remaining_seconds", 0)
            debit_done = result.get("debit_completed", False)
            outbox_rdy = result.get("outbox_ready", False)
            failure    = result.get("failure_reason", "")
            db_status  = result.get("status", "")
            display    = wf_state or db_status

            def fmt_seconds(s):
                if not s:
                    return "—"
                m, sec = divmod(int(s), 60)
                return f"{m}m {sec}s" if m else f"{sec}s"

            import textwrap
            st.markdown(textwrap.dedent(f"""
            <div class="result-box">
                <div class="result-key">Transfer ID</div>
                <div class="result-val">{st.session_state.status_transfer_id}</div>
                <div class="result-key">Workflow State</div>
                <div class="result-val">{status_badge(display)}</div>
{'<div class="result-key">Elapsed</div><div class="result-val">' + fmt_seconds(elapsed) + '</div>' if elapsed else ''}
{'<div class="result-key">Remaining</div><div class="result-val">' + fmt_seconds(remaining) + '</div>' if remaining else ''}
{'<div class="result-key">Debit Done</div><div class="result-val">✅ Yes</div>' if debit_done else ''}
{'<div class="result-key">Outbox Picked Up</div><div class="result-val">✅ Yes</div>' if outbox_rdy else ''}
{'<div class="result-key">Failure Reason</div><div class="result-val" style="color:#fc8181;">' + failure + '</div>' if failure else ''}
{'<div class="result-key">DB Status</div><div class="result-val">' + status_badge(db_status) + '</div>' if db_status and not wf_state else ''}
            </div>
            """), unsafe_allow_html=True)
        elif st.session_state.status_transfer_id and st.session_state.status_result is None and not st.session_state.cancel_result:
            # Only show "not found" if we actually tried a lookup (ID is set but result came back None)
            # We use a flag to avoid showing this on initial page load
            pass


    # ── All Gin Transfers ────────────────────────────────────────────────────
    st.markdown('<hr class="divider">', unsafe_allow_html=True)
    st.markdown('<div class="section-header">All Transfers — Gin Processor</div>', unsafe_allow_html=True)

    all_transfers = get(GIN_URL, "/transfers")
    all_accounts  = get(SPRING_BOOT_URL, "/accounts") or []
    account_id_map = {acc["id"]: acc["name"] for acc in all_accounts}

    if all_transfers is not None:
        if len(all_transfers) == 0:
            st.markdown('<div style="font-size:0.85rem;color:#555;padding:0.5rem 0;">No transfers found.</div>', unsafe_allow_html=True)
        else:
            # Stats row
            total   = len(all_transfers)
            pending = sum(1 for t in all_transfers if t.get("status", "").upper() in ("PENDING", "PROCESSING"))
            failed  = sum(1 for t in all_transfers if t.get("status", "").upper() == "FAILED")
            success = sum(1 for t in all_transfers if t.get("status", "").upper() == "SUCCESS")

            m1, m2, m3, m4 = st.columns(4)
            for col, label, value, color in [
                (m1, "Total",      total,   "#f0f0f0"),
                (m2, "Success",    success, "#c8f060"),
                (m3, "In-Flight",  pending, "#f5c842"),
                (m4, "Failed",     failed,  "#ff4d4d"),
            ]:
                with col:
                    st.markdown(f"""
                    <div class="metric-card" style="padding:0.9rem 1.2rem;">
                        <div class="metric-label">{label}</div>
                        <div class="metric-value" style="font-size:1.3rem;color:{color};">{value}</div>
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

                mode_colors = {"IMPS": "#c8f060", "NEFT": "#60b0f0", "RTGS": "#f5c842"}
                mode_color  = mode_colors.get(mode.upper(), "#888")

                st.markdown(f"""
                <div class="transfer-card">
                    <div style="display:flex;justify-content:space-between;align-items:center;">
                        <div style="flex:1;">
                            <div style="display:flex;align-items:center;gap:0.6rem;margin-bottom:0.35rem;">
                                <span style="font-size:0.72rem;font-weight:700;font-family:'JetBrains Mono',monospace;
                                             color:{mode_color};background:rgba(255,255,255,0.05);
                                             padding:0.15rem 0.55rem;border-radius:3px;border:1px solid rgba(255,255,255,0.08);">
                                    {mode}
                                </span>
                                {status_badge(status)}
                            </div>
                            <div style="font-size:0.85rem;color:#ccc;font-weight:500;">
                                {from_name}
                                <span style="color:#444;margin:0 0.4rem;">→</span>
                                {to_name}
                            </div>
                            <div style="font-size:0.7rem;color:#444;font-family:'JetBrains Mono',monospace;margin-top:0.2rem;">
                                {tid}
                            </div>
                        </div>
                        <div style="text-align:right;">
                            <div style="font-family:'JetBrains Mono',monospace;font-size:1rem;font-weight:500;color:#f0f0f0;">
                                {fmt_currency(amount)}
                            </div>
                            <div style="font-size:0.7rem;color:#444;margin-top:0.25rem;">
                                {fmt_date(created)}
                            </div>
                        </div>
                    </div>
                </div>
                """, unsafe_allow_html=True)
    else:
        st.warning("Could not reach Gin transfer processor.")


elif page == "Analytics":
    st.markdown('<div class="page-title">Analytics</div>', unsafe_allow_html=True)
    st.markdown('<div class="page-subtitle">Transaction statistics and high-value transfers</div>', unsafe_allow_html=True)

    summary    = get(FASTAPI_URL, "/analytics/summary")
    high_value = get(FASTAPI_URL, "/analytics/high-value")

    if summary:
        col1, col2, col3 = st.columns(3)

        with col1:
            st.markdown(f"""
            <div class="metric-card">
                <div class="metric-label">Total Transactions</div>
                <div class="metric-value">{summary.get('total_transactions', 0)}</div>
            </div>
            """, unsafe_allow_html=True)

        with col2:
            st.markdown(f"""
            <div class="metric-card">
                <div class="metric-label">Total Volume</div>
                <div class="metric-value">{fmt_currency(summary.get('total_volume', 0))}</div>
            </div>
            """, unsafe_allow_html=True)

        with col3:
            hv_count = high_value.get("count", 0) if high_value else 0
            st.markdown(f"""
            <div class="metric-card">
                <div class="metric-label">High Value (above 50,000)</div>
                <div class="metric-value">{hv_count}</div>
            </div>
            """, unsafe_allow_html=True)

        col1, col2 = st.columns(2)

        with col1:
            st.markdown('<div class="section-header">By Transfer Mode</div>', unsafe_allow_html=True)
            for item in summary.get("by_transfer_mode", []):
                st.markdown(f"""
                <div class="account-row">
                    <div style="display:flex;justify-content:space-between;align-items:center;">
                        <div class="account-name">{item.get('mode', '')}</div>
                        <div class="account-balance">{item.get('count', 0)} txns</div>
                    </div>
                </div>
                """, unsafe_allow_html=True)

        with col2:
            st.markdown('<div class="section-header">By Status</div>', unsafe_allow_html=True)
            for item in summary.get("by_status", []):
                st.markdown(f"""
                <div class="account-row">
                    <div style="display:flex;justify-content:space-between;align-items:center;">
                        <div>{status_badge(item.get('status', ''))}</div>
                        <div class="account-balance">{item.get('count', 0)}</div>
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
                        <div class="mono">{txn.get('id', '')}</div>
                        <div style="font-size:0.75rem;color:#444;margin-top:0.2rem;">{fmt_date(txn.get('createdAt', ''))}</div>
                    </div>
                    <div style="text-align:right;">
                        <div class="account-balance">{fmt_currency(txn.get('amount', 0))}</div>
                        <div style="margin-top:0.35rem;">{status_badge(txn.get('status', ''))}</div>
                    </div>
                </div>
            </div>
            """, unsafe_allow_html=True)
    else:
        st.info("No high value transfers found.")