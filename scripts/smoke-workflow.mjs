// Mutates only an explicitly selected disposable QA instance, never localhost:8080.
// QA_BASE_URL + QA_CREDENTIALS_FILE are required. Provision a fresh QA database first.
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { randomUUID } from 'node:crypto';
import { performance } from 'node:perf_hooks';

const base = process.env.QA_BASE_URL;
assert(
  base && process.env.QA_CREDENTIALS_FILE,
  'Set QA_BASE_URL and QA_CREDENTIALS_FILE for a disposable QA server.',
);
const target = new URL(base);
assert(
  ['localhost', '127.0.0.1', '[::1]'].includes(target.hostname),
  'QA smoke is restricted to a local disposable server.',
);
assert.notEqual(
  target.port || '80',
  '8080',
  'Port 8080 is reserved for the live demo; use a separate QA port.',
);
const exported = JSON.parse(await readFile(process.env.QA_CREDENTIALS_FILE, 'utf8'));
const employeeLogin = process.env.QA_EMPLOYEE_LOGIN || 'E0001';
const primary = exported.accounts.find((a) => a.login === employeeLogin && a.role === 'employee');
const peer = exported.accounts.find((a) => a.role === 'employee' && a.login !== employeeLogin);
const reviewer = exported.accounts.find(
  (a) => a.role === 'hr' && a.employee_id !== primary?.employee_id,
);
assert(
  primary && peer && reviewer,
  'Need two individual employee accounts and a non-self HR account in the private QA export.',
);
const timings = [];
const json = (body) => ({ method: 'POST', body: JSON.stringify(body) });
async function request(path, cookie, init = {}) {
  const start = performance.now();
  const headers = { ...(cookie ? { Cookie: cookie } : {}), ...init.headers };
  if (init.body && !(init.body instanceof FormData)) headers['Content-Type'] = 'application/json';
  const response = await fetch(`${base}/api${path}`, { ...init, headers });
  const bytes = new Uint8Array(await response.arrayBuffer());
  let body = null;
  if (response.headers.get('content-type')?.includes('application/json'))
    body = JSON.parse(new TextDecoder().decode(bytes));
  timings.push({ path, ms: Math.round(performance.now() - start) });
  return { response, body, bytes };
}
async function okay(path, cookie, init) {
  const result = await request(path, cookie, init);
  assert.equal(
    result.response.status,
    200,
    `${path}: ${result.response.status} ${JSON.stringify(result.body)}`,
  );
  return result.body;
}
async function login(account) {
  const { response } = await request(
    '/session',
    null,
    json({ login: account.login, password: account.password }),
  );
  assert.equal(response.status, 200, `Account login failed: ${account.login}`);
  assert.match(response.headers.get('set-cookie'), /HttpOnly/i);
  return response.headers.get('set-cookie').split(';')[0];
}
assert.equal((await request('/hr/submissions')).response.status, 401);
const employee = await login(primary);
const other = await login(peer);
const hr = await login(reviewer);
const id = primary.employee_id;
for (const path of [
  '/employees',
  `/employees/${peer.employee_id}`,
  '/hr/overview',
  '/hr/submissions',
])
  assert.equal((await request(path, employee)).response.status, 403, path);
const before = await okay(`/employees/${id}`, employee);
const baselineEXP = await okay(`/employees/${id}/experience`, employee);
assert.equal(before.employee.employee_id, id);
assert.equal((await okay('/session', other)).employee_id, peer.employee_id);
assert.equal(
  (await request(`/employees/${id}/completions`, employee, json({ event_id: 'EV_036' }))).response
    .status,
  410,
);
const recommendations = await okay(`/employees/${id}/recommendations`, employee);
assert(recommendations.items.length <= 3);
assert.equal(
  recommendations.mode,
  'rules',
  'Unset the real LLM key in the QA process before running the bulk smoke.',
);
for (const item of recommendations.items)
  assert.deepEqual(
    new Set(item.evidence.map((e) => e.factor)),
    new Set(['goal', 'gap', 'history']),
  );
const development = await okay(`/employees/${id}/development`, employee);
const preferred = recommendations.items.find((r) => r.event.event_id !== 'EV_036');
const module =
  development.items.find((m) => m.event.event_id === preferred?.event.event_id) ||
  development.items.find((m) => m.state === 'available' && m.event.event_id !== 'EV_036');
assert(
  module && module.state !== 'completed',
  'Fresh QA profile needs an unfinished nonrepeatable module.',
);
for (const locale of ['ru', 'kk', 'en']) {
  await okay('/me/preferences', employee, { ...json({ locale }), method: 'PATCH' });
  assert.equal((await okay('/session', employee)).locale, locale);
  const help = await okay(
    `/employees/${id}/assistant`,
    employee,
    json({ event_id: module.event.event_id, action: 'fifteen_minutes', locale }),
  );
  assert(help.benefit && help.first_step && help.application);
  assert.equal(help.mode, 'rules');
}
const enrollment = await okay(`/employees/${id}/modules/${module.event.event_id}/start`, employee, {
  method: 'POST',
});
const png = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=',
  'base64',
);
const firstKey = randomUUID();
function proofForm(key) {
  const form = new FormData();
  form.set('text', 'QA: demonstrated a practical work result and documented the tradeoffs.');
  form.set('url', 'https://example.com/qa-proof');
  form.set('request_key', key);
  form.append('files', new Blob([png], { type: 'image/png' }), 'synthetic-proof.png');
  return form;
}
const first = await okay(`/enrollments/${enrollment.enrollment.id}/submissions`, employee, {
  method: 'POST',
  body: proofForm(firstKey),
});
const duplicate = await okay(`/enrollments/${enrollment.enrollment.id}/submissions`, employee, {
  method: 'POST',
  body: proofForm(firstKey),
});
assert.equal(duplicate.submission.id, first.submission.id);
assert.deepEqual(
  (await okay(`/employees/${id}`, employee)).effective_skills,
  before.effective_skills,
);
assert.equal(
  (await okay(`/employees/${id}/experience`, employee)).total_exp,
  baselineEXP.total_exp,
);
const attachment = first.submission.attachments[0];
assert(attachment && !('storage_key' in attachment));
assert.equal((await request(`/attachments/${attachment.id}`, other)).response.status, 403);
const download = await request(`/attachments/${attachment.id}`, hr);
assert.equal(download.response.status, 200);
assert.deepEqual(Buffer.from(download.bytes), png);
assert.match(download.response.headers.get('content-disposition'), /^attachment/);
assert.equal(
  (
    await request(
      `/hr/submissions/${first.submission.id}/decision`,
      employee,
      json({ action: 'approve', request_key: randomUUID() }),
    )
  ).response.status,
  403,
);
const queue = await okay(
  `/hr/submissions?status=pending&employee_id=${encodeURIComponent(id)}`,
  hr,
);
assert(
  queue.pending_count >= 1 && queue.items.some((r) => r.submission.id === first.submission.id),
);
await okay(
  `/hr/submissions/${first.submission.id}/decision`,
  hr,
  json({
    action: 'return',
    comment: 'Explain your decision and one measurable result.',
    request_key: randomUUID(),
  }),
);
const revised = await okay(
  `/enrollments/${enrollment.enrollment.id}/submissions`,
  employee,
  json({
    text: 'QA revision: explained the decision, limitations and a measurable practical result.',
    url: '',
    request_key: randomUUID(),
  }),
);
assert.equal(revised.versions.length, 2);
const goals = await okay('/catalog/goals', employee);
const newGoal = goals.items.find((g) => g.target_role !== before.trajectory.goal.target_role);
if (newGoal)
  await okay(`/employees/${id}/goal`, employee, { ...json({ goal: newGoal }), method: 'PATCH' });
const approveKey = randomUUID();
const results = await Promise.all(
  Array.from({ length: 6 }, () =>
    okay(
      `/hr/submissions/${revised.submission.id}/decision`,
      hr,
      json({ action: 'approve', comment: 'QA evidence checked.', request_key: approveKey }),
    ),
  ),
);
assert(results.every((r) => r.submission.status === 'completed'));
const approved = results[0];
const afterEXP = await okay(`/employees/${id}/experience`, employee);
assert.equal(afterEXP.total_exp, baselineEXP.total_exp + enrollment.enrollment.config.reward_exp);
assert.equal(
  afterEXP.monthly_exp,
  baselineEXP.monthly_exp + enrollment.enrollment.config.reward_exp,
);
const after = await okay(`/employees/${id}`, employee);
assert(
  Object.keys(after.effective_skills).some(
    (skill) => after.effective_skills[skill] > (before.effective_skills[skill] || 0),
  ),
  'Approved module did not improve an expected skill.',
);
const publicBoard = await okay('/leaderboard', employee);
for (const entry of publicBoard.items)
  assert.deepEqual(Object.keys(entry).sort(), ['exp', 'full_name', 'rank']);
assert((await okay('/leaderboard', hr)).items.some((entry) => entry.employee_id === id));
const approval = approved.decisions.find((d) => d.action === 'approve');
assert(approval);
await okay(
  `/hr/approvals/${approval.id}/revoke`,
  hr,
  json({ comment: 'QA reversal confirms audit and compensation work.', request_key: randomUUID() }),
);
assert.equal(
  (await okay(`/employees/${id}/experience`, employee)).total_exp,
  baselineEXP.total_exp,
);
assert.deepEqual(
  (await okay(`/employees/${id}`, employee)).effective_skills,
  before.effective_skills,
);
await okay(`/employees/${id}/goal`, employee, {
  ...json({ goal: before.employee.career_goal }),
  method: 'PATCH',
});

const directory = (await okay('/employees', hr)).items;
const freshID = `JURY_QA_${Date.now()}`;
const imported = {
  ...before.employee,
  employee_id: freshID,
  full_name: 'Synthetic QA Profile',
  manager_id: null,
};
function importFiles(event = 'EV_036') {
  const form = new FormData();
  form.set(
    'employees',
    new Blob(
      [
        JSON.stringify({
          meta: { dataset: 'Career Quest', version: '1.0', as_of_date: before.as_of_date },
          employees: [imported],
        }),
      ],
      { type: 'application/json' },
    ),
    'employees.json',
  );
  form.set(
    'history',
    new Blob(
      [
        `record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\n${freshID}_RECORD,${freshID},${event},${before.as_of_date},,no_show,0,,,self\n`,
      ],
      { type: 'text/csv' },
    ),
    'history.csv',
  );
  return form;
}
const revision = (await okay('/health')).revision;
assert.equal(
  (await request('/hr/import', hr, { method: 'POST', body: importFiles('INVALID_EVENT') })).response
    .status,
  422,
);
assert.equal((await okay('/health')).revision, revision);
await okay('/hr/import', hr, { method: 'POST', body: importFiles() });
await okay('/hr/import', hr, { method: 'POST', body: importFiles() });
assert.equal((await okay('/employees', hr)).items.length, directory.length + 1);
assert.equal((await okay(`/employees/${freshID}/experience`, hr)).total_exp, 0);
for (const person of directory) {
  const profile = await okay(`/employees/${person.employee_id}`, hr);
  assert.equal(profile.employee.employee_id, person.employee_id);
}
const overview = await okay('/hr/overview', hr);
assert.equal(overview.employee_count, directory.length + 1);
const maxApiMs = Math.max(
  ...timings
    .filter((t) => !t.path.endsWith('/assistant') && !t.path.endsWith('/recommendations'))
    .map((t) => t.ms),
);
assert(maxApiMs < 2000, `Ordinary API exceeded 2 seconds: ${maxApiMs}ms`);
console.log(
  JSON.stringify(
    {
      result: 'passed',
      profilesChecked: directory.length,
      submissions: 2,
      concurrentApprovalRetries: 6,
      expectedAward: enrollment.enrollment.config.reward_exp,
      afterApprovalEXP: afterEXP.total_exp,
      afterReversalEXP: baselineEXP.total_exp,
      importedID: freshID,
      maxApiMs,
    },
    null,
    2,
  ),
);
