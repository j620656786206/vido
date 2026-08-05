import requests
import json

BASE_URL = "http://localhost:8090"
TIMEOUT = 30

def test_provider_key_save_resolves_secret_and_never_echoes_value():
    keys_endpoint = f"{BASE_URL}/api/v1/settings/keys"
    test_key = "sk-ant-api03-testsprite-verify-key-7f3a"
    masked_expected = test_key[:6] + chr(8230) + test_key[-4:]

    # Step 1: GET /api/v1/settings/keys and assert initial state
    resp = requests.get(keys_endpoint, timeout=TIMEOUT)
    assert resp.status_code == 200, f"Unexpected status code {resp.status_code} on initial GET"
    body = resp.json()
    assert body.get("success") is True, "Response success is not true on initial GET"
    data = body.get("data")
    assert data is not None, "No data field in response on initial GET"
    keys = data.get("keys")
    assert keys is not None, "No keys field in data on initial GET"
    assert isinstance(keys, dict), "Keys is not a dict on initial GET"
    expected_keys = {"claude", "tmdb", "openai"}
    actual_keys = set(keys.keys())
    assert actual_keys == expected_keys, f"Keys do not match expected: {actual_keys} vs {expected_keys}"
    for key_name in expected_keys:
        key_entry = keys[key_name]
        assert key_entry.get("configured") is False, f"{key_name} configured is not False initially"
        assert key_entry.get("source") == "none", f"{key_name} source is not 'none' initially"
        assert "masked" not in key_entry, f"{key_name} has 'masked' field initially, should not"

    # Step 2: PUT /api/v1/settings/keys with claude key
    try:
        put_payload = {"claude": test_key}
        resp = requests.put(keys_endpoint, json=put_payload, timeout=TIMEOUT)
        assert resp.status_code == 200, f"Unexpected status code {resp.status_code} on PUT update"
        body = resp.json()
        assert body.get("success") is True, "Response success is not true on PUT update"
        data = body.get("data")
        assert data is not None, "No data field on PUT update response"
        claude_entry = data.get("claude")
        assert claude_entry is not None, "No 'claude' key in data on PUT update response"
        assert claude_entry.get("configured") is True, "'claude' configured not True after PUT"
        assert claude_entry.get("source") == "secret", "'claude' source not 'secret' after PUT"
        masked = claude_entry.get("masked")
        assert masked == masked_expected, f"'claude' masked hint wrong: {masked}"
        # Assert full key does not appear anywhere in the response body string
        response_text = resp.text
        assert test_key not in response_text, "Full key appears in response body after PUT"

        # Step 3: GET again and assert secret-sourced state persists
        resp = requests.get(keys_endpoint, timeout=TIMEOUT)
        assert resp.status_code == 200, f"Unexpected status code {resp.status_code} on GET after PUT"
        body = resp.json()
        assert body.get("success") is True, "Response success is not true on GET after PUT"
        data = body.get("data")
        assert data is not None, "No data on GET after PUT"
        keys = data.get("keys")
        assert keys is not None, "No keys field on GET after PUT"
        claude_entry = keys.get("claude")
        assert claude_entry is not None, "No 'claude' in keys on GET after PUT"
        assert claude_entry.get("configured") is True, "'claude' configured not True on GET after PUT"
        assert claude_entry.get("source") == "secret", "'claude' source not 'secret' on GET after PUT"
        masked = claude_entry.get("masked")
        assert masked == masked_expected, f"'claude' masked hint wrong on GET after PUT: {masked}"
    finally:
        # Step 4 (cleanup): PUT {"claude": ""} to clear key and assert revert
        resp = requests.put(keys_endpoint, json={"claude": ""}, timeout=TIMEOUT)
        assert resp.status_code == 200, f"Unexpected status code {resp.status_code} on cleanup PUT"
        body = resp.json()
        assert body.get("success") is True, "Response success is not true on cleanup PUT"
        data = body.get("data")
        assert data is not None, "No data on cleanup PUT"
        claude_entry = data.get("claude")
        assert claude_entry is not None, "No 'claude' in data on cleanup PUT"
        assert claude_entry.get("configured") is False, "'claude' configured not False after cleanup"
        assert claude_entry.get("source") == "none", "'claude' source not 'none' after cleanup"
        assert "masked" not in claude_entry, "'claude' has 'masked' field after cleanup, should not"

test_provider_key_save_resolves_secret_and_never_echoes_value()