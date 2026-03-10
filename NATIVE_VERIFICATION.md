# Native Transaction Verification Guide

## Prerequisites

1. You have captured a real txid trace with `X_BROWSER_TRACE_TXID_FILE`
2. You have extracted a valid 58-character hex salt using `x harvest-txid`
3. You have configured the salt in `~/.config/x/config.yaml`

## Verification Steps

### Step 1: Configure Static Salt

```yaml
# ~/.config/x/config.yaml
browser:
  static_salt: "7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c"
```

### Step 2: Verify Doctor Output

```bash
./x doctor
```

Expected output:
```
transaction-id  ok      native-transaction-provider + static-salt
```

### Step 3: Run Wire Test (No Whitening - Default)

```bash
# Test with default (no whitening)
./x post "Native transport test #1 - no whitening"
```

**If it succeeds**: ✅ No whitening needed! Native transport is ready.

**If you get Error 344**: Whitening is required. Proceed to Step 4.

### Step 4: Enable Whitening (If Step 3 Failed)

Edit `internal/xapi/native_transport.go` and enable whitening:

```go
// In Generate() method, uncomment or enable:
buffer = p.whiten(buffer, salt)
```

Rebuild and test:

```bash
go build ./cmd/x
./x post "Native transport test #2 - with whitening"
```

**If it succeeds**: ✅ Whitening confirmed. Update the code to enable it by default.

**If it still fails**: The whitening pattern may be different. Check the deobfuscated JS for the exact XOR pattern.

## Salt Format Verification

Your salt should be:
- Exactly **58 hexadecimal characters** (0-9, a-f, A-F)
- Represents **29 bytes** of binary data
- No spaces, dashes, or special characters
- Example: `7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0c`

## Debugging Failed Tests

If the native post fails, capture the trace and compare:

```bash
# Enable tracing
export X_BROWSER_TRACE_TXID_FILE=/tmp/debug_txids.jsonl

# Run browser-based post (baseline)
./x post "Browser baseline test"

# Run native post (test)
./x post "Native transport test"

# Compare the generated txids
./x analyze-txid /tmp/debug_txids.jsonl
./x compare-txid /tmp/debug_txids.jsonl
```

Look for:
1. **Same salt**: Confirms static salt is working
2. **Different txids**: Normal - digest changes with timestamp
3. **Same length (70 bytes)**: Confirms structure is correct

## Expected Whitening Pattern

If whitening is active, the transformation should be:

```
Original:  [0x01][Digest (32 bytes)][Counter (8 bytes)][Salt (29 bytes)]
Whitened:  [0x01][Digest ⊕ Salt...][Counter ⊕ Salt...][Salt (unchanged)]
```

Where `⊕` is XOR and the salt is applied circularly to bytes 1-69.

## Success Criteria

✅ **Native transport working when:**
- Doctor shows `+ static-salt`
- Posts succeed without Error 344
- No browser automation needed

🎉 **You're ready for production!**
