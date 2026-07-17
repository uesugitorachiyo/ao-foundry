#!/usr/bin/env python3
import argparse
import copy
import hashlib
import json
import pathlib
import sys


ROOT = pathlib.Path(__file__).resolve().parents[1]


def read_json(path):
    with open(resolve(path), encoding="utf-8") as handle:
        return json.load(handle)


def write_json(path, value):
    target = resolve(path)
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(json.dumps(value, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def resolve(path):
    candidate = pathlib.Path(path)
    if candidate.is_absolute():
        return candidate
    return ROOT / candidate


def sha256_file(path):
    candidate = resolve(path)
    if not candidate.exists():
        return "0" * 64
    return hashlib.sha256(candidate.read_bytes()).hexdigest()


def valid_fixture(name):
    return read_json(f"examples/contract-fixtures/valid/{name}.json")


def source_artifact(name, path, schema_version, status="ready"):
    return {
        "name": name,
        "path": path,
        "schema_version": schema_version,
        "status": status,
        "sha256": sha256_file(path),
    }


def authority_boundaries(**extra):
    base = {
        "dry_run_only": True,
        "live_mutation_allowed": False,
        "mutates_repositories": False,
        "schedules_work": False,
        "executes_work": False,
        "approves_work": False,
        "provider_calls_allowed": False,
        "release_or_publish_allowed": False,
    }
    base.update(extra)
    return base


def governed_chain(args):
    if args.mutation_class == "low_risk_code":
        doc = {
            "schema_version": "ao.foundry.governed-live-mutation-dry-run-chain.v0.1",
            "status": "ready",
            "mode": "fixture_only_dry_run",
            "mutation_class": "low_risk_code",
            "current_proven_live_class": "test_only",
            "safe_to_request": True,
            "safe_to_execute": False,
            "exact_next_action": "build_low_risk_code_live_execution_gate",
            "source_artifacts": [
                source_artifact("atlas_classification", "examples/class-gate/atlas-classification.low-risk-code.json", "ao.atlas.mutation-classification.v0.1"),
                source_artifact("foundry_class_gate", "examples/class-gate/gate.dry-run.low-risk-code.json", "ao.foundry.class-gate.v0.1"),
                source_artifact("covenant_ticket", "examples/class-gate/covenant-ticket.low-risk-code.json", "covenant.low-risk-code-live-policy.v1", "approved"),
                source_artifact("forge_plan", "examples/class-gate/forge-plan.low-risk-code.json", "ao.forge.live-mutation-dry-run-plan.v0.1"),
                source_artifact("ao2_packet", "examples/class-gate/ao2-packet.low-risk-code.json", "ao2.live-mutation-dry-run-packet.v1"),
                source_artifact("rollback_proof", "examples/class-gate/rollback.passed.low-risk-code.json", "ao.foundry.mutation-class-rollback.v0.1", "passed"),
                source_artifact("sentinel_hold", "examples/class-gate/sentinel-hold.low-risk-code.json", "ao.sentinel.live-mutation-hold.v0.1", "clear"),
                source_artifact("promoter_ready", "examples/class-gate/promoter.ready.low-risk-code.json", "ao.promoter.live-mutation-boundary.v0.1", "passed"),
                source_artifact("command_readback", "examples/class-gate/command-readback.low-risk-code.json", "ao.command.live-mutation-status.v0.1"),
                source_artifact("test_only_success", "examples/class-gate/test-only-success.low-risk-code.json", "ao.foundry.test-only-success.v0.1"),
            ],
            "readiness_assessment": {
                "includes_atlas": True,
                "includes_covenant": True,
                "includes_forge": True,
                "includes_ao2": True,
                "includes_rollback": True,
                "includes_sentinel": True,
                "includes_promoter": True,
                "includes_command": True,
                "live_mutation_performed": False,
                "ungated_live_mutation_claim": False,
            },
            "authority_boundaries": authority_boundaries(),
        }
    else:
        doc = valid_fixture("foundry-governed-live-mutation-dry-run-chain-v0.1")
    write_json(pathlib.Path(args.out) / "summary.json", doc)
    print(f"governed_live_mutation_chain={args.out}")
    return 0


def low_risk_gate(args):
    checks = []
    first = ""

    def add(name, status):
        nonlocal first
        checks.append({"name": name, "status": status, "summary": name})
        if status != "passed" and not first:
            first = name

    add("low_risk_code_dry_run_chain", "passed")
    if not args.atlas_blueprint_import or not args.atlas_status:
        add("atlas_blueprint_import", "blocked")
        next_step = "collect_atlas_blueprint_import_readback"
    else:
        add("atlas_blueprint_import", "passed")
        if not args.live_policy_evidence or "weak" in pathlib.Path(args.live_policy_evidence).name:
            add("live_policy_evidence", "blocked")
            next_step = "collect_low_risk_code_live_policy_evidence"
        else:
            add("live_policy_evidence", "passed")
            if not args.bounded_packet_proof or "other-branch" in resolve(args.bounded_packet_proof).read_text(encoding="utf-8"):
                add("bounded_packet_enforcement", "blocked")
                next_step = "collect_forge_ao2_bounded_packet_enforcement_proof"
            else:
                add("bounded_packet_enforcement", "passed")
                add("sentinel_low_risk_live_verdict", "passed")
                add("promoter_low_risk_live_verdict", "passed")
                add("command_live_readback", "passed")
                next_step = "request_low_risk_code_live_rehearsal"

    ready = first == ""
    doc = {
        "schema_version": "ao.foundry.low-risk-code-live-rehearsal-gate.v0.1",
        "status": "ready" if ready else "blocked",
        "mutation_class": "low_risk_code",
        "safe_to_request": True,
        "safe_to_execute": ready,
        "first_failing_check": first,
        "exact_next_step": next_step,
        "checks": checks,
        "authority_boundaries": authority_boundaries(
            creates_branch=False,
            creates_worktree=False,
            opens_pr=False,
            merges_pr=False,
            multi_repo_mutation_allowed=False,
            complex_repo_mutation_allowed=False,
            fully_unsupervised_complex_mutation_claimed=False,
        ),
    }
    write_json(args.out, doc)
    print(f"low_risk_code_live_rehearsal_gate={doc['status']}")
    return 0


def live_mutation_rollup(args):
    doc = valid_fixture("foundry-live-mutation-readiness-rollup-v0.1")
    doc["source_chain"]["path"] = args.chain
    doc["source_chain"]["sha256"] = sha256_file(args.chain)
    write_json(args.out, doc)
    print("live_mutation_readiness_rollup=ready")
    return 0


def approval_gate(args):
    ready = "approved" in pathlib.Path(args.ticket).name
    doc = valid_fixture("foundry-live-docs-approval-gate-v0.1")
    if not ready:
        doc["status"] = "blocked"
        doc["safe_to_execute"] = False
        doc["approval_state"] = "pending"
        doc["first_failing_check"] = "approval_state"
    write_json(args.out, doc)
    print(f"live_docs_approval_gate={doc['status']}")
    return 0


def worktree_isolation(args):
    doc = valid_fixture("foundry-worktree-isolation-proof-v0.1")
    doc["candidate_path"] = args.candidate
    doc["candidate_sha256"] = sha256_file(args.candidate)
    name = pathlib.Path(args.candidate).name
    if "dirty" in name:
        doc["status"] = "blocked"
        doc["first_failing_check"] = "clean_worktree"
    elif "reused" in name:
        doc["status"] = "blocked"
        doc["first_failing_check"] = "reuse_block"
    write_json(args.out, doc)
    return 0 if doc["status"] == "ready" else 1


def live_docs_prepare(args):
    doc = valid_fixture("foundry-live-docs-worktree-prepare-v0.1")
    name = pathlib.Path(args.candidate).name
    if "invalid" in args.approval_gate:
        doc["status"] = "blocked"
        doc["can_start_docs_only_pr_rehearsal"] = False
        doc["first_failing_check"] = "approval_gate"
    elif "dirty" in name:
        doc["status"] = "blocked"
        doc["can_start_docs_only_pr_rehearsal"] = False
        doc["first_failing_check"] = "clean_worktree"
    elif "reused" in name:
        doc["status"] = "blocked"
        doc["can_start_docs_only_pr_rehearsal"] = False
        doc["first_failing_check"] = "reuse_block"
    elif "wrong-branch" in name:
        doc["status"] = "blocked"
        doc["can_start_docs_only_pr_rehearsal"] = False
        doc["first_failing_check"] = "branch_isolation"
    elif "unsafe" in name:
        doc["status"] = "blocked"
        doc["can_start_docs_only_pr_rehearsal"] = False
        doc["first_failing_check"] = "docs_only_path_plan"
    write_json(args.out, doc)
    return 0 if doc["status"] == "ready" else 1


def live_docs_rollback(args):
    doc = valid_fixture("foundry-live-docs-rollback-execution-rehearsal-v0.1")
    name = pathlib.Path(args.candidate).name
    if "unsafe" in name:
        doc["status"] = "blocked"
        doc["rollback_verified"] = False
        doc["first_failing_check"] = "docs_only_target"
    elif "missing" in name:
        doc["status"] = "blocked"
        doc["rollback_verified"] = False
        doc["first_failing_check"] = "rollback_patch_present"
    write_json(args.out, doc)
    return 0 if doc["status"] == "ready" else 1


def approved_live_docs_chain(args):
    doc = valid_fixture("foundry-approved-live-docs-dry-run-chain-v0.1")
    write_json(pathlib.Path(args.out) / "summary.json", doc)
    print(f"approved_live_docs_dry_run_chain={args.out}")
    return 0


def live_docs_pr_gate(args):
    doc = valid_fixture("foundry-live-docs-pr-rehearsal-gate-v0.1")
    if not args.approval_artifact:
        doc["status"] = "blocked"
        doc["safe_to_execute"] = False
        doc["first_failing_check"] = "approval_artifact"
        doc["exact_next_step"] = "request_operator_approval"
    write_json(args.out, doc)
    return 0


def first_live_docs_rollup(args):
    gate = read_json(args.pr_gate)
    doc = valid_fixture("foundry-first-live-docs-readiness-rollup-v0.1")
    if gate.get("status") != "ready":
        doc["status"] = "blocked"
        doc["safe_to_execute"] = False
        doc["explicit_operator_approval_required"] = True
        doc["exact_next_step"] = "request_operator_approval"
        doc["first_failing_check"] = gate.get("first_failing_check", "approval_artifact")
    write_json(args.out, doc)
    return 0


def rollback_rehearsal(args):
    doc = valid_fixture("foundry-live-mutation-rollback-rehearsal-v0.1")
    name = pathlib.Path(args.candidate).name
    if "missing" in name:
        doc["status"] = "blocked"
        doc["first_failing_check"] = "rollback_patch_present"
    elif "unsafe" in name:
        doc["status"] = "blocked"
        doc["first_failing_check"] = "authority_boundaries"
    write_json(args.out, doc)
    return 0 if doc["status"] == "ready" else 1


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--script", required=True)
    parser.add_argument("--out")
    parser.add_argument("--mutation-class", default="docs_only_multi_file")
    parser.add_argument("--chain")
    parser.add_argument("--atlas-blueprint-import")
    parser.add_argument("--atlas-status")
    parser.add_argument("--live-policy-evidence")
    parser.add_argument("--bounded-packet-proof")
    parser.add_argument("--sentinel-verdict")
    parser.add_argument("--promoter-verdict")
    parser.add_argument("--command-readback")
    parser.add_argument("--request")
    parser.add_argument("--ticket")
    parser.add_argument("--candidate")
    parser.add_argument("--approval-gate")
    parser.add_argument("--approval-artifact", default="")
    parser.add_argument("--pr-gate")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    handlers = {
        "governed-live-mutation-dry-run-chain.sh": governed_chain,
        "low-risk-code-live-rehearsal-gate.sh": low_risk_gate,
        "live-mutation-readiness-rollup.sh": live_mutation_rollup,
        "live-docs-approval-gate.sh": approval_gate,
        "live-mutation-worktree-isolation-proof.sh": worktree_isolation,
        "live-docs-worktree-prepare.sh": live_docs_prepare,
        "live-docs-rollback-execution-rehearsal.sh": live_docs_rollback,
        "approved-live-docs-dry-run-chain.sh": approved_live_docs_chain,
        "live-docs-pr-rehearsal-gate.sh": live_docs_pr_gate,
        "first-live-docs-readiness-rollup.sh": first_live_docs_rollup,
        "live-mutation-rollback-rehearsal.sh": rollback_rehearsal,
    }
    handler = handlers.get(args.script)
    if handler is None:
        print(f"unsupported fixture script: {args.script}", file=sys.stderr)
        return 2
    return handler(args)


if __name__ == "__main__":
    raise SystemExit(main())
