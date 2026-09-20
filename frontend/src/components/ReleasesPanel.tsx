import { useState, type FormEvent } from "react";
import type { ChangeView } from "../api/changes";
import { newIdempotencyKey, type GateResult, type ReleaseView } from "../api/releases";
import type { TopologyEnvironment } from "../api/topology";
import type { ReleasesController } from "../state/useReleases";
import { StatusBadge } from "./StatusBadge";

interface ReleasesPanelProps {
  controller: ReleasesController;
  changes: ChangeView[];
  environments: TopologyEnvironment[];
}

const lines = (value: string): string[] => value.split("\n").map((line) => line.trim()).filter(Boolean);

function GateTable({ gates, testid }: { gates: GateResult[]; testid: string }) {
  return (
    <table className="topology-table gate-table" data-testid={testid}>
      <thead><tr><th>Gate</th><th>Change</th><th>Result</th><th>Detail</th></tr></thead>
      <tbody>
        {gates.map((gate, index) => (
          <tr key={`${gate.gate}-${gate.change_id ?? ""}-${index}`} data-testid={`${testid}-${gate.gate}${gate.change_id ? `-${gate.change_id}` : ""}`}>
            <td><code>{gate.gate}</code></td>
            <td>{gate.change_id ? <code>{gate.change_id}</code> : <span className="muted">—</span>}</td>
            <td><StatusBadge status={gate.status} /></td>
            <td>{gate.detail}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

// ---------------------------------------------------------------- create

function CreateReleaseForm({ changes, environments, busy, onCancel, onSubmit }: { changes: ChangeView[]; environments: TopologyEnvironment[]; busy: boolean; onCancel: () => void; onSubmit: ReleasesController["create"] }) {
  const promotable = changes.filter((view) => view.change.status === "CANDIDATE_ACCEPTED" || (view.change.candidates ?? []).some((candidate) => candidate.status === "ACCEPTED_FOR_PROMOTION"));
  const others = changes.filter((view) => !promotable.includes(view));
  const [selected, setSelected] = useState<string[]>([]);
  const [target, setTarget] = useState(promotable[0]?.decision?.target_environment.id ?? "staging");
  const [reason, setReason] = useState("");
  const toggle = (id: string) => setSelected((previous) => (previous.includes(id) ? previous.filter((entry) => entry !== id) : [...previous, id]));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const ok = await onSubmit({ reason: reason.trim(), change_ids: selected, target_environment_id: target });
    if (ok) onCancel();
  };
  const selectable = environments.filter((environment) => environment.status === "ACTIVE");
  return (
    <form className="topology-form" data-testid="release-form-create" onSubmit={(event) => void submit(event)}>
      <p className="eyebrow">NEW RELEASE · bundle one or more Changes; one blocked Change blocks the whole bundle</p>
      <fieldset className="option-list">
        <legend>Changes to promote</legend>
        {promotable.length === 0 && <p className="muted">No Change has an accepted candidate yet.</p>}
        {promotable.map((view) => (
          <label key={view.change.change_id} className="checkbox-line">
            <input type="checkbox" name={`change-${view.change.change_id}`} checked={selected.includes(view.change.change_id)} onChange={() => toggle(view.change.change_id)} />
            <code>{view.change.change_id}</code> {view.change.title} <StatusBadge status={view.change.status} />
          </label>
        ))}
        {others.map((view) => (
          <label key={view.change.change_id} className="checkbox-line muted" title="No accepted candidate; including it will block the bundle (UI-13)">
            <input type="checkbox" name={`change-${view.change.change_id}`} checked={selected.includes(view.change.change_id)} onChange={() => toggle(view.change.change_id)} />
            <code>{view.change.change_id}</code> {view.change.title} <StatusBadge status={view.change.status} /> <span className="muted">no accepted candidate</span>
          </label>
        ))}
      </fieldset>
      <div className="topology-form-grid">
        <label>Target environment
          <select name="target_environment_id" value={target} onChange={(event) => setTarget(event.target.value)}>
            {selectable.map((environment) => <option key={environment.id} value={environment.id}>{environment.display_name} ({environment.type}{environment.protection ? ", blocked in P0" : ""})</option>)}
          </select>
        </label>
        <label>Reason<input name="reason" value={reason} onChange={(event) => setReason(event.target.value)} required /></label>
      </div>
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy || selected.length === 0}>Open release and run promotion gate</button>
        <button type="button" className="button-secondary" onClick={onCancel} disabled={busy}>Cancel</button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------- detail

function ReleaseDetail({ view, controller }: { view: ReleaseView; controller: ReleasesController }) {
  const { release, live_gates: liveGates, staleness } = view;
  const manifest = release.manifest;
  const [build, setBuild] = useState({ image_digest: "", image_ref: "", build_id: "", source_commit: manifest.changes[0]?.head_commit ?? "", includes_commits: manifest.changes.map((entry) => entry.head_commit).join("\n"), evidence_ref: "", reason: "" });
  const [approval, setApproval] = useState({ actor: "", role: "release_manager", rationale: "" });
  const [deploy, setDeploy] = useState({ idempotency_key: newIdempotencyKey(), operation_id: "", revision: "", service_url: "", deployed_digest: manifest.build?.image_digest ?? "", deployed_target_ref: manifest.environment.target_ref, smoke_status: "PASS", smoke_ref: "", reason: "" });
  const closed = release.status === "PROMOTED" || release.status === "REJECTED";
  const blocked = liveGates.some((gate) => gate.status === "BLOCKED" || gate.status === "STALE");
  const canBuild = !closed;
  const canApprove = !closed && !blocked && manifest.build !== null && !(release.approval?.decision === "APPROVED");
  const canDeploy = !closed && release.approval?.decision === "APPROVED" && !staleness.stale;

  return (
    <div className="change-detail" data-testid="release-detail">
      <div className="topology-change-heading">
        <code>{release.release_id}</code>
        <StatusBadge status={release.status} />
        <span className="muted">{manifest.transition} · target <code>{manifest.environment.target_ref}</code> · topology v{manifest.environment.topology_version} · config <code title={manifest.environment.config_hash}>{manifest.environment.config_hash.slice(7, 19)}</code> · manifest <code title={manifest.manifest_hash}>{manifest.manifest_hash.slice(7, 19)}</code></span>
      </div>
      {staleness.stale && <div className="topology-inline-error" data-testid="release-stale" role="alert"><strong>STALE</strong> — {staleness.reason}</div>}

      <div className="decision-card">
        <p className="eyebrow">MANIFEST · changes in this release</p>
        <table className="topology-table">
          <thead><tr><th>Change</th><th>Decision</th><th>Work order</th><th>Candidate</th><th>Commit</th><th>Review</th></tr></thead>
          <tbody>
            {manifest.changes.map((entry) => (
              <tr key={entry.change_id} data-testid={`release-change-${entry.change_id}`}>
                <td><code>{entry.change_id}</code> {entry.title}</td>
                <td>{entry.decision_id ? `${entry.decision_id} v${entry.decision_version}` : <span className="muted">∅</span>}</td>
                <td>{entry.work_order_id || <span className="muted">∅</span>}</td>
                <td>{entry.candidate_id ? `${entry.candidate_id} by ${entry.candidate_accepted_by}` : <span className="reason">none accepted</span>}</td>
                <td><code>{entry.head_commit || "∅"}</code></td>
                <td>{entry.review_gate || <span className="muted">∅</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
        <p className="topology-reason"><span className="muted">build:</span> {manifest.build ? <>{manifest.build.image_digest} <span className="muted">from {manifest.build.source_commit} · {manifest.build.build_id || "no build id"} · {manifest.build.evidence_ref || "no evidence ref"}</span></> : <span className="reason">no image digest yet</span>}</p>
      </div>

      <div>
        <p className="eyebrow">PROMOTION GATE · live</p>
        <GateTable gates={liveGates} testid="release-gates" />
      </div>

      {canBuild && (
        <form className="topology-form" data-testid="release-build-form" onSubmit={(event) => { event.preventDefault(); void controller.build(release.release_id, { reason: build.reason.trim(), image_digest: build.image_digest.trim(), image_ref: build.image_ref.trim() || undefined, build_id: build.build_id.trim() || undefined, source_commit: build.source_commit.trim(), includes_commits: lines(build.includes_commits), evidence_ref: build.evidence_ref.trim() || undefined }); }}>
          <p className="eyebrow">BUILD EVIDENCE · the digest becomes the release identity; it must be built from the accepted commits</p>
          <div className="topology-form-grid">
            <label className="span-2">Image digest (sha256:…)<input name="image_digest" value={build.image_digest} onChange={(event) => setBuild({ ...build, image_digest: event.target.value })} required placeholder="sha256:…" /></label>
            <label>Source commit<input name="source_commit" value={build.source_commit} onChange={(event) => setBuild({ ...build, source_commit: event.target.value })} required /></label>
            <label>Build id<input name="build_id" value={build.build_id} onChange={(event) => setBuild({ ...build, build_id: event.target.value })} /></label>
            <label>Includes commits (one per line)<textarea name="includes_commits" rows={2} value={build.includes_commits} onChange={(event) => setBuild({ ...build, includes_commits: event.target.value })} /></label>
            <label>Evidence ref<input name="evidence_ref" value={build.evidence_ref} onChange={(event) => setBuild({ ...build, evidence_ref: event.target.value })} placeholder="cloud build log url" /></label>
            <label className="span-2">Reason<input name="reason" value={build.reason} onChange={(event) => setBuild({ ...build, reason: event.target.value })} required /></label>
          </div>
          <div className="topology-form-actions"><button type="submit" className="button-primary" disabled={controller.busy}>Record build</button></div>
        </form>
      )}

      {release.approval && (
        <div className="decision-card" data-testid="release-approval">
          <div className="topology-change-heading">
            <strong>Release approval</strong>
            <StatusBadge status={release.approval.decision} />
            <span className="muted">{release.approval.actor} ({release.approval.role}) at {release.approval.at} · binds manifest {release.approval.manifest_hash.slice(7, 19)} · expires {release.approval.expires_at}</span>
          </div>
          <p className="topology-reason">{release.approval.reason}</p>
          {release.approval.waiver && <p className="reason">{release.approval.waiver}</p>}
        </div>
      )}

      {canApprove && (
        <form className="topology-form" data-testid="release-approval-form" onSubmit={(event) => { event.preventDefault(); void controller.approve(release.release_id, { actor: approval.actor.trim(), role: approval.role.trim(), decision: "APPROVE", rationale: approval.rationale.trim() }); }}>
          <p className="eyebrow">RELEASE APPROVAL · third human decision, bound to manifest {manifest.manifest_hash.slice(7, 19)}</p>
          <div className="topology-form-grid">
            <label>Approver<input name="actor" value={approval.actor} onChange={(event) => setApproval({ ...approval, actor: event.target.value })} required placeholder="must not be the run starter unless policy allows" /></label>
            <label>Role<input name="role" value={approval.role} onChange={(event) => setApproval({ ...approval, role: event.target.value })} /></label>
            <label className="span-2">Rationale<input name="rationale" value={approval.rationale} onChange={(event) => setApproval({ ...approval, rationale: event.target.value })} required /></label>
          </div>
          <div className="topology-form-actions">
            <button type="submit" className="button-primary" disabled={controller.busy}>Approve release</button>
            <button type="button" className="button-secondary" disabled={controller.busy} onClick={() => void controller.approve(release.release_id, { actor: approval.actor.trim(), role: approval.role.trim(), decision: "REJECT", rationale: approval.rationale.trim() })}>Reject release</button>
          </div>
        </form>
      )}
      {!closed && blocked && <p className="reason" data-testid="release-blocked-note">Promotion is blocked; the approval control stays hidden until every gate passes. No partial promotion.</p>}

      {canDeploy && (
        <form className="topology-form" data-testid="release-deployment-form" onSubmit={(event) => { event.preventDefault(); void controller.deployment(release.release_id, { reason: deploy.reason.trim(), idempotency_key: deploy.idempotency_key, operation_id: deploy.operation_id.trim() || undefined, revision: deploy.revision.trim(), service_url: deploy.service_url.trim() || undefined, deployed_digest: deploy.deployed_digest.trim(), deployed_target_ref: deploy.deployed_target_ref.trim(), smoke: { status: deploy.smoke_status, evidence_ref: deploy.smoke_ref.trim() } }); }}>
          <p className="eyebrow">DEPLOYMENT RECORD · what actually landed (from scripts/record-promotion.sh or Cloud Run describe)</p>
          <div className="topology-form-grid">
            <label>Idempotency key<input name="idempotency_key" value={deploy.idempotency_key} onChange={(event) => setDeploy({ ...deploy, idempotency_key: event.target.value })} required /></label>
            <label>Operation id<input name="operation_id" value={deploy.operation_id} onChange={(event) => setDeploy({ ...deploy, operation_id: event.target.value })} /></label>
            <label>Revision<input name="revision" value={deploy.revision} onChange={(event) => setDeploy({ ...deploy, revision: event.target.value })} required placeholder="service-00003-abc" /></label>
            <label>Service URL<input name="service_url" value={deploy.service_url} onChange={(event) => setDeploy({ ...deploy, service_url: event.target.value })} /></label>
            <label className="span-2">Deployed digest<input name="deployed_digest" value={deploy.deployed_digest} onChange={(event) => setDeploy({ ...deploy, deployed_digest: event.target.value })} required /></label>
            <label className="span-2">Deployed target ref<input name="deployed_target_ref" value={deploy.deployed_target_ref} onChange={(event) => setDeploy({ ...deploy, deployed_target_ref: event.target.value })} required /></label>
            <label>Smoke result
              <select name="smoke_status" value={deploy.smoke_status} onChange={(event) => setDeploy({ ...deploy, smoke_status: event.target.value })}>
                <option value="PASS">PASS</option><option value="FAIL">FAIL</option>
              </select>
            </label>
            <label>Smoke evidence ref<input name="smoke_ref" value={deploy.smoke_ref} onChange={(event) => setDeploy({ ...deploy, smoke_ref: event.target.value })} /></label>
            <label className="span-2">Reason<input name="reason" value={deploy.reason} onChange={(event) => setDeploy({ ...deploy, reason: event.target.value })} required /></label>
          </div>
          <div className="topology-form-actions"><button type="submit" className="button-primary" disabled={controller.busy}>Record deployment and verify</button></div>
        </form>
      )}

      {release.deployments.slice().reverse().map((attempt) => (
        <div className="decision-card" key={attempt.attempt_id} data-testid={`release-attempt-${attempt.attempt_id}`}>
          <div className="topology-change-heading">
            <code>{attempt.attempt_id}</code>
            <StatusBadge status={attempt.outcome} />
            <span className="muted">{attempt.revision || "no revision"} · {attempt.deployed_digest.slice(0, 19)} → {attempt.deployed_target_ref} · key {attempt.idempotency_key} · {attempt.recorded_at}</span>
          </div>
          <GateTable gates={attempt.gates} testid={`attempt-gates-${attempt.attempt_id}`} />
        </div>
      ))}

      {release.receipt && (
        <div className="decision-card receipt-card" data-testid="release-receipt">
          <div className="topology-change-heading">
            <strong>Release Receipt</strong>
            <code>{release.receipt.receipt_id}</code>
            <StatusBadge status={release.receipt.status} />
            <span className="muted">issued {release.receipt.issued_at} · hash <code>{release.receipt.receipt_hash.slice(0, 26)}</code></span>
          </div>
          <ul className="reason-list">
            <li>Transition {release.receipt.transition} → <code>{release.receipt.environment.target_ref}</code> (topology v{release.receipt.environment.topology_version}, config {release.receipt.environment.config_hash.slice(7, 19)})</li>
            {release.receipt.changes.map((entry) => <li key={entry.change_id}>{entry.change_id} · {entry.decision_id} v{entry.decision_version} · {entry.work_order_id} · {entry.candidate_id} · commit <code>{entry.head_commit}</code> · review {entry.review_gate}</li>)}
            <li>Image <code>{release.receipt.build.image_digest}</code> built from <code>{release.receipt.build.source_commit}</code></li>
            <li>Approved by {release.receipt.approval.actor} ({release.receipt.approval.role}) at {release.receipt.approval.at}</li>
            <li>Revision <code>{release.receipt.deployment.revision}</code> · {release.receipt.deployment.service_url || "no url"} · smoke {release.receipt.deployment.smoke?.status ?? "∅"} · operation {release.receipt.deployment.operation_id || "∅"}</li>
          </ul>
        </div>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- panel

export function ReleasesPanel({ controller, changes, environments }: ReleasesPanelProps) {
  const { status, releases, selected, error, busy } = controller;
  const [creating, setCreating] = useState(false);
  return (
    <section className="panel topology-panel" aria-labelledby="releases-heading" data-testid="releases-panel">
      <div className="section-heading">
        <div>
          <p className="eyebrow">STAGING PROMOTION</p>
          <h2 id="releases-heading">Releases</h2>
        </div>
        <span className="muted">Bundle → gate → build digest → third human approval → deployment record → receipt. Same digest is necessary, not sufficient.</span>
      </div>
      {status === "LOADING" && <p className="muted" data-testid="releases-loading"><span className="spinner" aria-hidden="true" />Loading releases…</p>}
      {error && (
        <div className="topology-inline-error" data-testid="releases-error" role="alert">
          <strong>{error.code}</strong> — {error.message}
          {status === "ERROR" && <button type="button" className="button-secondary" onClick={controller.reload}>Reload releases</button>}
        </div>
      )}
      {status !== "LOADING" && (
        <div className="changes-layout">
          <div className="changes-list">
            {releases.length === 0 && <p className="muted" data-testid="releases-empty">No release opened yet.</p>}
            {releases.map((view) => (
              <button type="button" key={view.release.release_id} className={`project-card${controller.selectedID === view.release.release_id ? " project-card-selected" : ""}`} data-testid={`release-card-${view.release.release_id}`} onClick={() => { setCreating(false); controller.select(view.release.release_id); }}>
                <span className="project-card-title">{view.release.release_id}</span>
                <span className="project-card-id">{view.release.manifest.changes.map((entry) => entry.change_id).join(" + ")} → {view.release.manifest.environment.environment_id}</span>
                <span className="project-card-meta"><StatusBadge status={view.release.status} />{view.release.receipt && <StatusBadge status={view.release.receipt.receipt_id} />}</span>
              </button>
            ))}
            {!creating && <button type="button" className="button-primary" disabled={busy} onClick={() => { controller.select(null); setCreating(true); }}>New release</button>}
          </div>
          <div className="changes-detail">
            {creating && <CreateReleaseForm changes={changes} environments={environments} busy={busy} onCancel={() => setCreating(false)} onSubmit={controller.create} />}
            {!creating && selected && <ReleaseDetail key={`${selected.release.release_id}-${selected.release.manifest.manifest_hash}-${selected.release.approval?.decision ?? ""}-${selected.release.deployments.length}`} view={selected} controller={controller} />}
            {!creating && !selected && releases.length > 0 && <p className="muted">Select a release.</p>}
          </div>
        </div>
      )}
    </section>
  );
}
