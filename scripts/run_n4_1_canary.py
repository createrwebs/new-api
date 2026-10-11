#!/usr/bin/env python3
import zlib, struct, base64, json, time, subprocess, requests, os, bcrypt, hashlib

BASE_URL = "http://127.0.0.1:3000"

def db_exec(sql):
    cmd = ["docker", "exec", "-i", "postgres", "psql", "-U", "root", "-d", "new-api", "-At"]
    res = subprocess.run(cmd, input=sql, text=True, capture_output=True, check=True)
    return res.stdout.strip()

def make_png(w, h):
    raw_data = bytearray()
    for y in range(h):
        raw_data.append(0) # filter none
        for x in range(w):
            if w//4 <= x <= 3*w//4 and h//4 <= y <= 3*h//4:
                raw_data.extend([40, 120, 220, 255]) # blue product
            else:
                raw_data.extend([0, 0, 0, 0])
    def chunk(tag, data):
        return struct.pack(">I", len(data)) + tag + data + struct.pack(">I", zlib.crc32(tag + data) & 0xffffffff)
    ihdr = struct.pack(">IIBBBBB", w, h, 8, 6, 0, 0, 0)
    idat = zlib.compress(bytes(raw_data))
    return b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", ihdr) + chunk(b"IDAT", idat) + chunk(b"IEND", b"")

def wait_for_wallet_quota(session, base_url, headers, expected_quota, max_wait=12):
    t_start = time.time()
    while time.time() - t_start < max_wait:
        r = session.get(f"{base_url}/api/user/self", headers=headers)
        if r.status_code == 200:
            q = r.json().get("data", {}).get("quota")
            if q == expected_quota:
                return q
        time.sleep(0.5)
    r = session.get(f"{base_url}/api/user/self", headers=headers)
    return r.json().get("data", {}).get("quota")

def main():
    print("=====================================================================")
    print("TORA STUDIO QUEUE N4.1 — PRODUCTION CANARY & PROBE VERIFICATION")
    print("=====================================================================")

    # 1. Probes
    print("\n--- 1. Probing Core Endpoints & Infrastructure ---")
    r_root = requests.get(f"{BASE_URL}/", timeout=5)
    print(f"GET / -> status={r_root.status_code}")
    assert r_root.status_code == 200, "Root probe failed"

    r_status = requests.get(f"{BASE_URL}/api/status", timeout=5)
    print(f"GET /api/status -> status={r_status.status_code}")
    assert r_status.status_code == 200, "API health status failed"

    r_tools = requests.get(f"{BASE_URL}/api/studio/tools", timeout=5)
    print(f"GET /api/studio/tools -> status={r_tools.status_code}")
    assert r_tools.status_code == 200, "Studio tools failed"

    r_templates = requests.get(f"{BASE_URL}/api/studio/native/seller-templates", timeout=5)
    print(f"GET /api/studio/native/seller-templates -> status={r_templates.status_code}")
    assert r_templates.status_code == 200, "Seller templates failed"
    templates_list = r_templates.json().get("data", [])
    print(f"  -> Found {len(templates_list)} canonical seller templates")

    # DB & Redis checks
    pg_status = subprocess.run(["docker", "exec", "postgres", "pg_isready"], capture_output=True, text=True).stdout.strip()
    redis_ping = subprocess.run(["docker", "exec", "redis", "redis-cli", "ping"], capture_output=True, text=True).stdout.strip()
    print(f"PostgreSQL probe: {pg_status}")
    print(f"Redis probe: {redis_ping}")

    # 2. Authenticate Ephemeral Canary User
    print("\n--- 2. Setting Up Authenticated Canary User ---")
    test_user = "canary_n4_1_auditor"
    password_raw = "AuditN41@2026!Secure"
    password_hash = bcrypt.hashpw(password_raw.encode("utf-8"), bcrypt.gensalt()).decode("utf-8")
    now = int(time.time())
    initial_quota = 5000000 # 5,000 Credits

    db_exec(f"DELETE FROM users WHERE username = '{test_user}';")
    user_id_raw = db_exec(f"""
        INSERT INTO users (username, password, display_name, role, status, quota, auth_version, created_at)
        VALUES ('{test_user}', '{password_hash}', 'N4.1 Auditor', 10, 1, {initial_quota}, 1, {now})
        RETURNING id;
    """)
    user_id = int(user_id_raw.split()[0])
    print(f"  -> Created canary user: id={user_id}, initial_quota={initial_quota} ({initial_quota//1000} Credits)")

    session = requests.Session()
    login_resp = session.post(f"{BASE_URL}/api/user/login", json={
        "username": test_user,
        "password": password_raw
    })
    assert login_resp.status_code == 200, f"Login failed: {login_resp.text}"
    token = login_resp.json()["data"]["access_token"]
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }

    # Verify wallet before
    r_self_before = session.get(f"{BASE_URL}/api/user/self", headers=headers)
    wallet_before = r_self_before.json()["data"]["quota"]
    print(f"  -> Verified initial wallet: {wallet_before} quota ({wallet_before//1000} Credits)")
    assert wallet_before == initial_quota

    # 3. Product Factory Canary (Section 5)
    print("\n--- 3. Running Controlled Seller Factory Production Canary ---")
    # Quote
    quote_payload = {
        "input_count": 1,
        "enable_2x_upscale": False
    }
    r_pf_quote = session.post(f"{BASE_URL}/api/studio/native/product-factory/quote", json=quote_payload, headers=headers)
    assert r_pf_quote.status_code == 200, f"PF quote failed: {r_pf_quote.text}"
    q_data = r_pf_quote.json().get("data", {})
    quote_id = q_data.get("quote_id")
    pricing_ver = q_data.get("pricing_version")
    req_credits = q_data.get("total_credits")
    req_quota = q_data.get("total_quota")
    print(f"  -> Obtained Quote: quote_id={quote_id}, version={pricing_ver}, credits={req_credits}, quota={req_quota}")
    assert req_credits == 7, "Expected 7 credits for 1-item base product factory"

    # Execution
    cutout_png = make_png(200, 200)
    cutout_b64 = base64.b64encode(cutout_png).decode("ascii")
    batch_payload = {
        "batch_id": f"canary_n41_pf_{now}",
        "items": [{
            "index": 0,
            "cutout_png_base64": cutout_b64,
            "original_name": "clean_product_canary.png"
        }],
        "selected_templates": ["shopee_standard", "lazada_hd", "tiktok_shop"],
        "bg_preset": "PURE_WHITE",
        "shadow_preset": "MARKETPLACE",
        "brand_hex": "#112233",
        "include_zip": True
    }
    t0 = time.time()
    r_batch = session.post(f"{BASE_URL}/api/studio/native/product-factory/v2/batch", json=batch_payload, headers=headers)
    pf_duration_ms = int((time.time() - t0) * 1000)
    assert r_batch.status_code == 200, f"PF batch failed: {r_batch.text}"
    batch_res = r_batch.json().get("data", {}).get("result", {})
    zip_pkg = batch_res.get("zip_package", {})
    print(f"  -> PF Batch Succeeded in {pf_duration_ms}ms: success_items={batch_res.get('success_items')}, zip_size={zip_pkg.get('file_size')} bytes")
    assert batch_res.get("success_items") == 1

    # Verify wallet after Product Factory (accounting for write-behind batch update interval)
    print("  -> Awaiting DB batch update flush for Product Factory quota...")
    wallet_mid = wait_for_wallet_quota(session, BASE_URL, headers, wallet_before - 7000, max_wait=12)
    pf_charged_quota = wallet_before - wallet_mid
    print(f"  -> Wallet after PF: {wallet_mid} quota (delta: {pf_charged_quota} quota = {pf_charged_quota//1000} Credits)")
    assert pf_charged_quota == 7000, f"Expected 7000 quota deducted, got {pf_charged_quota}"

    # 4. Object Cleanup Canary (Section 6)
    print("\n--- 4. Running Controlled Object Cleanup Interactive Session Canary ---")
    # Quote
    quote_oc_payload = {
        "tool_id": "object-cleanup",
        "execution_class": "NATIVE_BROWSER"
    }
    r_oc_quote = session.post(f"{BASE_URL}/api/studio/native/quote", json=quote_oc_payload, headers=headers)
    assert r_oc_quote.status_code == 200, f"OC quote failed: {r_oc_quote.text}"
    oc_quote_data = r_oc_quote.json().get("data", {})
    oc_credits = oc_quote_data.get("credits") or oc_quote_data.get("estimated_credits")
    oc_quota = oc_quote_data.get("quota") or oc_quote_data.get("estimated_quota")
    print(f"  -> Object Cleanup Quote: tool={oc_quote_data.get('tool_id')}, credits={oc_credits}, quota={oc_quota}")
    assert oc_credits == 3, f"Expected 3 credits for object-cleanup, got {oc_credits}"
    assert oc_quota == 3000, f"Expected 3000 quota for object-cleanup, got {oc_quota}"

    # Start Session
    src_bytes = b"synthetic_product_photo_for_cleanup_canary"
    src_hash = hashlib.sha256(src_bytes).hexdigest()
    session_start_payload = {
        "source_hash": src_hash
    }
    r_oc_start = session.post(f"{BASE_URL}/api/studio/native/object-cleanup/session", json=session_start_payload, headers=headers)
    assert r_oc_start.status_code == 200, f"OC session start failed: {r_oc_start.text}"
    oc_session = r_oc_start.json().get("data", {}).get("session", {})
    session_id = oc_session.get("session_id")
    print(f"  -> Session created: session_id={session_id}, source_hash={oc_session.get('source_hash')[:12]}..., max_exports={oc_session.get('max_exports')}")
    assert oc_session.get("paid_credits") == 3

    # Check wallet deduction for Object Cleanup session activation (awaiting batch update flush)
    print("  -> Awaiting DB batch update flush for Object Cleanup session quota...")
    wallet_oc = wait_for_wallet_quota(session, BASE_URL, headers, wallet_mid - 3000, max_wait=12)
    oc_charged_quota = wallet_mid - wallet_oc
    print(f"  -> Wallet after OC start: {wallet_oc} quota (delta: {oc_charged_quota} quota = {oc_charged_quota//1000} Credits)")
    assert oc_charged_quota == 3000, f"Expected 3000 quota deducted for 3-credit session, got {oc_charged_quota}"

    # Validate session with matching source hash
    r_val_match = session.post(f"{BASE_URL}/api/studio/native/object-cleanup/session/validate", json={
        "session_id": session_id,
        "source_hash": src_hash
    }, headers=headers)
    assert r_val_match.status_code == 200
    assert r_val_match.json().get("success") is True
    print("  -> PASS: Session validated with matching source hash")

    # Validate session with mismatched source hash (Anti-arbitrage binding)
    r_val_mismatch = session.post(f"{BASE_URL}/api/studio/native/object-cleanup/session/validate", json={
        "session_id": session_id,
        "source_hash": "different_image_hash_attacker_attempt"
    }, headers=headers)
    assert r_val_mismatch.status_code == 400
    assert r_val_mismatch.json().get("success") is False
    print("  -> PASS: Source hash binding enforced (mismatched image rejected with 400)")

    # Record export within session (Fair Zero-Credit Retry / Export)
    r_export = session.post(f"{BASE_URL}/api/studio/native/object-cleanup/session/export", json={
        "session_id": session_id
    }, headers=headers)
    assert r_export.status_code == 200
    print("  -> Export recorded successfully")

    # Verify wallet remains unchanged after export within session (NO DOUBLE CHARGE)
    r_self_after_exp = session.get(f"{BASE_URL}/api/user/self", headers=headers)
    wallet_after_exp = r_self_after_exp.json()["data"]["quota"]
    print(f"  -> Wallet after export: {wallet_after_exp} quota (delta: {wallet_oc - wallet_after_exp})")
    assert wallet_after_exp == wallet_oc, "Double charge detected during export within active session!"
    print("  -> PASS: Zero additional credits charged for export within active session")

    # Cleanup test user
    db_exec(f"DELETE FROM users WHERE id = {user_id};")
    print("  -> Cleaned up canary test user")

    print("\n=====================================================================")
    print("ALL N4.1 CANARIES & SECURITY BINDINGS VERIFIED IN PRODUCTION")
    print("=====================================================================")

    evidence = {
        "timestamp": int(time.time()),
        "probes": {
            "root": r_root.status_code,
            "status": r_status.status_code,
            "tools": r_tools.status_code,
            "seller_templates": len(templates_list),
            "postgres": pg_status,
            "redis": redis_ping
        },
        "seller_factory_canary": {
            "workflow_id": f"canary_n41_pf_{now}",
            "quote_id": quote_id,
            "pricing_version": pricing_ver,
            "wallet_before": wallet_before,
            "credits_charged": 7,
            "quota_charged": 7000,
            "wallet_after": wallet_mid,
            "zip_file_size": zip_pkg.get("file_size"),
            "templates": ["shopee_standard", "lazada_hd", "tiktok_shop"],
            "shadow": "MARKETPLACE",
            "background": "PURE_WHITE",
            "duration_ms": pf_duration_ms,
            "result": "SUCCESS"
        },
        "object_cleanup_canary": {
            "session_id": session_id,
            "paid_credits": 3,
            "paid_quota": 3000,
            "source_hash_binding": "VERIFIED_ENFORCED",
            "zero_credit_export": "VERIFIED_NO_DOUBLE_CHARGE",
            "status": "VERIFIED_PRODUCTION"
        }
    }

    with open("/home/ubuntu/n4_1_canary_evidence.json", "w") as f:
        json.dump(evidence, f, indent=2)
    print("Evidence written to /home/ubuntu/n4_1_canary_evidence.json")

if __name__ == "__main__":
    main()
