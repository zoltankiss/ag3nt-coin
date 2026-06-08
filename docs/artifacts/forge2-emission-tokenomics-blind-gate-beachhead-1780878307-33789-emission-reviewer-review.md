# Emission Reviewer Review

Run id: `forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789`
Reviewer: `emission-reviewer`
Candidate: `protocol-miner-a`
Commit reviewed: `2c55815fe7716a19631ab07e8c6affdb7c68887b`
Reviewed at: `2026-06-08T01:10:00Z`

## Verdict

`merge`

The private GitHub artifact-check behavior is correct for the `0.5.0-beta.2`
beachhead. The client now detects pinned `zoltankiss/ag3nt-coin` GitHub blob
artifact URIs, fetches exact bytes through authenticated `gh api` with the raw
Accept header, reports `access_method: "gh"`, and preserves the existing public
HTTPS and local smoke-test behavior.

## Verification

```text
$ bun test client/src/ag3nt.test.ts
bun test v1.3.12 (700fc117)

client/src/ag3nt.test.ts:
(pass) artifact URI validation > local artifact override does not suppress known bad GitHub repo checks [0.29ms]
(pass) artifact URI validation > local artifact override still permits local smoke-test URIs [0.03ms]
(pass) artifact URI validation > local artifact override lets artifact-check fetch local http smoke-test URIs [0.05ms]
(pass) artifact URI validation > github blob artifact parser preserves pinned ref and path [0.07ms]
(pass) artifact URI validation > github blob artifact parser ignores non-blob github URLs [0.02ms]

 5 pass
 0 fail
 5 expect() calls
Ran 5 tests across 1 file. [26.00ms]
```

```text
$ bun --check client/src/ag3nt.ts client/src/ag3nt.test.ts
<no stdout/stderr; exit 0>
```

```text
$ gh repo view zoltankiss/ag3nt-coin --json nameWithOwner,visibility,url
{"nameWithOwner":"zoltankiss/ag3nt-coin","url":"https://github.com/zoltankiss/ag3nt-coin","visibility":"PRIVATE"}
```

```text
$ gh api -H "Accept: application/vnd.github.raw" 'repos/zoltankiss/ag3nt-coin/contents/docs/artifacts/forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789-protocol-miner-a-artifact-check.md?ref=2c55815fe7716a19631ab07e8c6affdb7c68887b' | shasum -a 256
7daf518a9c6337e8bf2bad2f4fb9237f285af0ca2d73195fb134a31dfb5c4f9e  -
```

```text
$ bun src/cli.ts artifact-check 'https://github.com/zoltankiss/ag3nt-coin/blob/2c55815fe7716a19631ab07e8c6affdb7c68887b/docs/artifacts/forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789-protocol-miner-a-artifact-check.md' 7daf518a9c6337e8bf2bad2f4fb9237f285af0ca2d73195fb134a31dfb5c4f9e
{
  "ok": true,
  "uri": "https://github.com/zoltankiss/ag3nt-coin/blob/2c55815fe7716a19631ab07e8c6affdb7c68887b/docs/artifacts/forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789-protocol-miner-a-artifact-check.md",
  "fetch_uri": "gh api repos/zoltankiss/ag3nt-coin/contents/docs/artifacts/forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789-protocol-miner-a-artifact-check.md?ref=2c55815fe7716a19631ab07e8c6affdb7c68887b",
  "access_method": "gh",
  "expected_sha256": "7daf518a9c6337e8bf2bad2f4fb9237f285af0ca2d73195fb134a31dfb5c4f9e",
  "actual_sha256": "7daf518a9c6337e8bf2bad2f4fb9237f285af0ca2d73195fb134a31dfb5c4f9e",
  "bytes": 1159
}
```

## Award Recommendation

```json
{
  "recommended_award_amount": 25,
  "award_class": "code_fix",
  "rationale": "This is a narrow client behavior fix with focused tests and a live private-repo artifact-check smoke. It directly removes beta.2 protocol-use ambiguity around private GitHub evidence without broadening local-only artifact behavior."
}
```

Residual note: branch names containing slashes would not round-trip through the
simple blob parser, but this run requires commit-pinned artifact URIs, so that
does not block the beta.2 beachhead behavior.
