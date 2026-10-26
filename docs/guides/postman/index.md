# Postman — AI Readiness Report

Scored with **`postman spec ai-readiness ./openapi.yaml`** (Postman CLI 1.60.0) against the git-registered spec (`inputType: FILE_PATH`).

[:material-file-chart: Open full HTML report](reports/ai-readiness.html){ .md-button .md-button--primary target=_blank }
[:material-code-json: Download JSON](reports/ai-readiness.json){ .md-button target=_blank }

## Headline

| Metric | Value |
| --- | --- |
| Readiness | **80 / 100 — Excellent** |
| Confidence | high *(0 signals `unknown`)* |
| Doc coverage | high *(adjustment −1)* |
| Complexity | 2 / 10 — Simple |
| Endpoints | 20 |
| Schemas | 6 |

!!! note "Why the score moved from 90 to 80 after we improved the spec"
    We documented credential acquisition, added response examples, and exposed rate-limit headers (`X-RateLimit-*`). That resolved both previously-`unknown` signals, so **confidence rose from `medium` to `high`**. But the readiness *score* fell 90 → 80: the rubric folds rate limiting into **complexity** (`rate_limiting: complex`, complexity 1→2/10), and readiness tracks inversely. In other words, better documentation made the API's real complexity *visible* — the score didn't get worse, our measurement of it got honest.

## Recommendations

Addressed in the spec: ✅ credential acquisition documented · ✅ response examples added · ✅ rate limiting exposed. Remaining, and largely inherent to the API's shape:

- **Scope the surface.** 20 endpoints is large; point an agent at a focused subset, or split the spec into smaller domains.
- **Simplify parameters.** Reduce parameter count or document each parameter's role — agents select parameters from descriptions.

## Full report

The command's own HTML report is served as a static asset and embedded below (passthrough rendering).

<iframe
  src="reports/ai-readiness.html"
  title="OpenAPI Petstore — AI Readiness report"
  loading="lazy"
  style="width:100%; height:820px; border:1px solid var(--md-default-fg-color--lightest); border-radius:8px; background:#fff;">
</iframe>

*Prefer a standalone tab? [Open the report directly](reports/ai-readiness.html){ target=_blank }.*
