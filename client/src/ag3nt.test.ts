import { afterEach, describe, expect, test } from "bun:test";
import { existsSync, readFileSync, unlinkSync } from "fs";
import { artifactFetchUri, assertContributionAwardRecipient, assertExternallyFetchableArtifactUri, contributionAwardResult, createGateTemplate, githubBlobArtifact } from "./ag3nt";

const originalAllowLocal = process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI;

afterEach(() => {
  if (originalAllowLocal === undefined) {
    delete process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI;
  } else {
    process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI = originalAllowLocal;
  }
});

describe("gate template", () => {
  test("stdout omits settlement secrets while private file keeps them", () => {
    const slug = `gate-template-secret-test-${process.pid}`;
    const publicPath = `${slug}.public-gate.md`;
    const privatePath = `${slug}.private-gate-secret.json`;
    try {
      const result = createGateTemplate(slug, "Y,N,N,Y,N", 5, "fixed-secret-salt");

      expect(JSON.stringify(result)).not.toContain("Y,N,N,Y,N");
      expect(JSON.stringify(result)).not.toContain("fixed-secret-salt");
      expect(result.settle_command).toBe(
        "ag3nt gate-settle <gate_id> <gold_answer_from_private_file> <gold_salt_from_private_file>",
      );

      const privateJson = JSON.parse(readFileSync(privatePath, "utf8"));
      expect(privateJson.gold_answer).toBe("Y,N,N,Y,N");
      expect(privateJson.gold_salt).toBe("fixed-secret-salt");
      expect(privateJson.settle_command).toBe("ag3nt gate-settle <gate_id> Y,N,N,Y,N fixed-secret-salt");
    } finally {
      if (existsSync(publicPath)) unlinkSync(publicPath);
      if (existsSync(privatePath)) unlinkSync(privatePath);
    }
  });

  test("rejects gold answer with the wrong question count before writing files", () => {
    const slug = `gate-template-count-test-${process.pid}`;
    const publicPath = `${slug}.public-gate.md`;
    const privatePath = `${slug}.private-gate-secret.json`;
    try {
      expect(() => createGateTemplate(slug, "Y,N,N,Y", 5, "fixed-secret-salt")).toThrow(
        "gold_answer must contain exactly 5 comma-separated Y/N values",
      );
      expect(existsSync(publicPath)).toBe(false);
      expect(existsSync(privatePath)).toBe(false);
    } finally {
      if (existsSync(publicPath)) unlinkSync(publicPath);
      if (existsSync(privatePath)) unlinkSync(privatePath);
    }
  });

  test("rejects non-binary gold answer values before writing files", () => {
    const slug = `gate-template-binary-test-${process.pid}`;
    const publicPath = `${slug}.public-gate.md`;
    const privatePath = `${slug}.private-gate-secret.json`;
    try {
      expect(() => createGateTemplate(slug, "Y,N,maybe,Y,N", 5, "fixed-secret-salt")).toThrow(
        "gold_answer values must be canonical Y or N",
      );
      expect(existsSync(publicPath)).toBe(false);
      expect(existsSync(privatePath)).toBe(false);
    } finally {
      if (existsSync(publicPath)) unlinkSync(publicPath);
      if (existsSync(privatePath)) unlinkSync(privatePath);
    }
  });
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

describe("contribution award preflight", () => {
  test("requires explicit reviewed contributor address", () => {
    expect(() => assertContributionAwardRecipient("agnt1anchor", "agnt1contributor", "")).toThrow(
      "requires --contributor-address",
    );
  });

  test("rejects recipient mismatch against reviewed contributor", () => {
    expect(() =>
      assertContributionAwardRecipient("agnt1anchor", "agnt1wrong", "agnt1contributor"),
    ).toThrow("does not match reviewed contributor");
  });

  test("rejects accidental self-awards without explicit override", () => {
    expect(() => assertContributionAwardRecipient("agnt1anchor", "agnt1anchor", "agnt1anchor")).toThrow(
      "recipient matches the signing anchor",
    );
  });

  test("requires review evidence for founder-authored awards", () => {
    expect(() => assertContributionAwardRecipient("agnt1anchor", "agnt1anchor", "agnt1anchor", true)).toThrow(
      "requires --review-evidence-uri",
    );
  });

  test("rejects founder-authored metadata on non-anchor awards", () => {
    expect(() =>
      assertContributionAwardRecipient(
        "agnt1anchor",
        "agnt1contributor",
        "agnt1contributor",
        true,
        "https://github.com/zoltankiss/ag3nt-coin/pull/1#review",
      ),
    ).toThrow("founder-authored contribution-award requires recipient to match the signing anchor");
  });

  test("allows reviewed founder-authored awards with explicit metadata", () => {
    expect(() =>
      assertContributionAwardRecipient(
        "agnt1anchor",
        "agnt1anchor",
        "agnt1anchor",
        true,
        "https://github.com/zoltankiss/ag3nt-coin/pull/1#review",
      ),
    ).not.toThrow();
  });

  test("allows awards to a distinct contributor", () => {
    expect(() => assertContributionAwardRecipient("agnt1anchor", "agnt1contributor", "agnt1contributor")).not.toThrow();
  });

  test("success output includes recipient binding metadata", () => {
    expect(
      contributionAwardResult(
        { id: "7", txhash: "ABC" },
        "agnt1anchor",
        "agnt1contributor",
        "3",
        "agnt1contributor",
      ),
    ).toEqual({
      ok: true,
      id: "7",
      anchor: "agnt1anchor",
      recipient: "agnt1contributor",
      contributor: "agnt1contributor",
      recipient_binding: true,
      founder_authored: false,
      review_evidence_uri: "",
      amount: "3",
      txhash: "ABC",
    });
  });

  test("success output preserves founder-authored review metadata", () => {
    expect(
      contributionAwardResult(
        { id: "8", txhash: "DEF" },
        "agnt1anchor",
        "agnt1anchor",
        "5",
        "agnt1anchor",
        true,
        "https://github.com/zoltankiss/ag3nt-coin/pull/1#review",
      ),
    ).toMatchObject({
      contributor: "agnt1anchor",
      recipient_binding: true,
      founder_authored: true,
      review_evidence_uri: "https://github.com/zoltankiss/ag3nt-coin/pull/1#review",
    });
  });
});
