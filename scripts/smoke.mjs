// Run against a disposable server/state only. Creates one extra synthetic profile.
// QA_BASE_URL, QA_EMPLOYEE_PASSWORD, QA_HR_PASSWORD are required; no real LLM needed.
import assert from 'node:assert/strict';
import { performance } from 'node:perf_hooks';
const base = process.env.QA_BASE_URL;
assert(
  base && process.env.QA_EMPLOYEE_PASSWORD && process.env.QA_HR_PASSWORD,
  'Set QA_BASE_URL and both QA_*_PASSWORD variables for a disposable test instance.',
);
const timings = [];
async function request(path, cookie, init = {}) {
  const started = performance.now();
  const headers = { ...(cookie ? { Cookie: cookie } : {}), ...init.headers };
  if (init.body && !(init.body instanceof FormData)) headers['Content-Type'] = 'application/json';
  const response = await fetch(`${base}/api${path}`, { ...init, headers });
  timings.push({ path, ms: Math.round(performance.now() - started) });
  const body = await response.json().catch(() => null);
  return { response, body };
}
async function login(role, password) {
  const { response } = await request('/session', null, {
    method: 'POST',
    body: JSON.stringify({ role, password }),
  });
  assert.equal(response.status, 200, `Login ${role}`);
  return response.headers.get('set-cookie').split(';')[0];
}
const employee = await login('employee', process.env.QA_EMPLOYEE_PASSWORD);
const hr = await login('hr', process.env.QA_HR_PASSWORD);
for (const path of ['/employees', '/employees/E0002', '/hr/overview'])
  assert.equal((await request(path, employee)).response.status, 403);
const before = (await request('/employees/E0001', employee)).body;
const recs = (await request('/employees/E0001/recommendations', employee)).body;
assert(recs.items.length > 0 && recs.items.length <= 3);
for (const item of recs.items)
  assert.deepEqual(
    new Set(item.evidence.map((e) => e.factor)),
    new Set(['goal', 'gap', 'history']),
  );
const completion = {
  method: 'POST',
  body: JSON.stringify({ event_id: recs.items[0].event.event_id }),
};
const completed = (await request('/employees/E0001/completions', employee, completion)).body;
assert(completed.changed && completed.profile.trajectory.progress > before.trajectory.progress);
const again = (await request('/employees/E0001/completions', employee, completion)).body;
assert.equal(again.changed, false);
assert.deepEqual(again.profile.effective_skills, completed.profile.effective_skills);
const directory = (await request('/employees', hr)).body.items;
assert(directory.length >= 200);
const crossRole = (await request('/employees/E0004', hr)).body;
assert.equal(crossRole.trajectory.goal.target_role, 'Product Manager');
const freshID = 'JURY_QA_001';
const imported = {
  ...before.employee,
  employee_id: freshID,
  full_name: 'Jury Demo Profile',
  manager_id: null,
};
const json = JSON.stringify({
  meta: { dataset: 'Career Quest', version: '1.0', as_of_date: before.as_of_date },
  employees: [imported],
});
function files(event = 'EV_036') {
  const form = new FormData();
  form.set('employees', new Blob([json], { type: 'application/json' }), 'employees.json');
  form.set(
    'history',
    new Blob(
      [
        `record_id,employee_id,event_id,date,due_date,status,completion_pct,score,feedback_rating,assigned_by\nJURY_QA_RECORD,${freshID},${event},2026-09-25,,no_show,0,,,self\n`,
      ],
      { type: 'text/csv' },
    ),
    'history.csv',
  );
  return form;
}
const revisionBefore = (await request('/health')).body.revision;
assert.equal(
  (await request('/hr/import', hr, { method: 'POST', body: files('INVALID_EVENT') })).response
    .status,
  422,
);
assert.equal((await request('/health')).body.revision, revisionBefore);
assert.equal(
  (await request('/hr/import', hr, { method: 'POST', body: files() })).response.status,
  200,
);
const countAfterImport = (await request('/employees', hr)).body.items.length;
assert.equal(
  (await request('/hr/import', hr, { method: 'POST', body: files() })).response.status,
  200,
);
assert.equal((await request('/employees', hr)).body.items.length, countAfterImport);
assert.equal((await request(`/employees/${freshID}/recommendations`, hr)).response.status, 200);
const overview = (await request('/hr/overview', hr)).body;
assert.equal(overview.employee_count, countAfterImport);
assert.equal(overview.event_count, 40);
assert.equal(overview.skill_count, 60);
for (const person of directory) {
  const { response, body } = await request(`/employees/${person.employee_id}/recommendations`, hr);
  assert.equal(response.status, 200);
  assert(body.items.length <= 3);
  if (!body.items.length) assert(body.empty_reason);
}
const maxMs = Math.max(...timings.map((t) => t.ms));
assert(maxMs < 2000, `Offline API exceeded 2s: ${maxMs}ms`);
console.log(
  JSON.stringify(
    {
      result: 'passed',
      profilesChecked: directory.length,
      progressBefore: before.trajectory.progress,
      progressAfter: completed.profile.trajectory.progress,
      importedID: freshID,
      maxApiMs: maxMs,
      withoutNextStep: overview.without_next_step.length,
    },
    null,
    2,
  ),
);
