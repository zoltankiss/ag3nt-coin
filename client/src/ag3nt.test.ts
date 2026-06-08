import { afterEach, describe, expect, test } from "bun:test";
import { artifactFetchUri, assertExternallyFetchableArtifactUri, githubBlobArtifact } from "./ag3nt";

const originalAllowLocal = process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI;

afterEach(() => {
  if (originalAllowLocal === undefined) {
    delete process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI;
  } else {
    process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI = originalAllowLocal;
  }
});

describe("artifact URI validation", () => {
  test("local artifact override does not suppress known bad GitHub repo checks", () => {
    process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI = "1";

    expect(() =>
      assertExternallyFetchableArtifactUri(
        "https://github.com/zoltankiss/agnt-coin/blob/main/docs/gate-v1-pr-review-beta.md",
        "payload_uri",
      ),
    ).toThrow("known bad GitHub artifact repo 'zoltankiss/agnt-coin'");
  });

  test("local artifact override still permits local smoke-test URIs", () => {
    process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI = "1";

    expect(() =>
      assertExternallyFetchableArtifactUri("http://127.0.0.1:4312/artifacts/payload.json"),
    ).not.toThrow();
  });

  test("local artifact override lets artifact-check fetch local http smoke-test URIs", () => {
    process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI = "1";

    expect(artifactFetchUri("http://127.0.0.1:4312/artifacts/payload.json")).toBe(
      "http://127.0.0.1:4312/artifacts/payload.json",
    );
  });

  test("github blob artifact parser preserves pinned ref and path", () => {
    expect(
      githubBlobArtifact(
        "https://github.com/zoltankiss/ag3nt-coin/blob/835544fa6a66010c69d2ad168d2a843087d9bccd/docs/gate-v1-pr-review-beta.md",
      ),
    ).toEqual({
      owner: "zoltankiss",
      repo: "ag3nt-coin",
      ref: "835544fa6a66010c69d2ad168d2a843087d9bccd",
      path: "docs/gate-v1-pr-review-beta.md",
    });
  });

  test("github blob artifact parser ignores non-blob github URLs", () => {
    expect(githubBlobArtifact("https://github.com/zoltankiss/ag3nt-coin/pull/12")).toBeNull();
  });
});
