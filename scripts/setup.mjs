import { readFile, writeFile, mkdir, access } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { randomBytes } from 'node:crypto';
import { parseEnv } from 'node:util';
import { root, run, finished } from './process.mjs';

try {
  const args = process.argv.slice(2);
  const index = args.indexOf('--data-dir');
  if (index < 0 || !args[index + 1])
    throw new Error('Usage: npm run setup -- --data-dir "<starter kit directory>"');
  const source = resolve(args[index + 1]);
  const prepared = [];
  for (const name of ['skills.json', 'employees.json', 'events.json', 'activity_history.csv']) {
    const alias = name.replace(/\.(json|csv)$/, '(1).$1');
    let bytes;
    for (const candidate of [name, alias]) {
      try {
        bytes = await readFile(join(source, candidate));
        break;
      } catch (error) {
        if (error.code !== 'ENOENT') throw error;
      }
    }
    if (!bytes) throw new Error(`Missing ${name} or ${alias} in ${source}`);
    if (name.endsWith('.json')) JSON.parse(bytes.toString('utf8').replace(/^\uFEFF/, ''));
    prepared.push([name, bytes]);
  }
  await mkdir(join(root, 'data/seed'), { recursive: true });
  await mkdir(join(root, 'data/runtime'), { recursive: true });
  for (const [name, bytes] of prepared) await writeFile(join(root, 'data/seed', name), bytes);
  let envText;
  try {
    envText = await readFile(join(root, '.env'), 'utf8');
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
    envText = await readFile(join(root, '.env.example'), 'utf8');
  }
  const env = parseEnv(envText);
  function setDefault(key, value) {
    if (env[key]) return;
    const line = `${key}=${value}`;
    const pattern = new RegExp(`^${key}=.*$`, 'm');
    // A callback keeps literal '$' characters in externally supplied values intact.
    envText = pattern.test(envText)
      ? envText.replace(pattern, () => line)
      : `${envText.trimEnd()}\n${line}\n`;
    env[key] = value;
  }
  setDefault('DEMO_EMPLOYEE_PASSWORD', randomBytes(8).toString('hex'));
  setDefault('DEMO_HR_PASSWORD', randomBytes(8).toString('hex'));
  setDefault('POSTGRES_DB', 'careerquest');
  setDefault('POSTGRES_USER', 'careerquest');
  setDefault('POSTGRES_PASSWORD', randomBytes(24).toString('hex'));
  setDefault('POSTGRES_PORT', '5433');
  setDefault(
    'DATABASE_URL',
    `postgresql://${encodeURIComponent(env.POSTGRES_USER)}:${encodeURIComponent(env.POSTGRES_PASSWORD)}@127.0.0.1:${env.POSTGRES_PORT}/${encodeURIComponent(env.POSTGRES_DB)}?sslmode=disable`,
  );
  await writeFile(join(root, '.env'), envText, { mode: 0o600 });
  await finished(
    run('go', ['run', './backend/cmd/server'], {
      env: {
        ...process.env,
        VALIDATE_ONLY: '1',
      },
    }),
  );
  let hasLock = false;
  try {
    await access(join(root, 'package-lock.json'));
    hasLock = true;
  } catch {
    /* First bootstrap. */
  }
  await finished(run('npm', [hasLock ? 'ci' : 'install']));
  console.log(
    '\nSetup complete. Start PostgreSQL: docker compose up -d postgres. Then run npm run demo.',
  );
  console.log(
    'Existing database data and configured .env values were preserved. Seed and .env are excluded from Git.',
  );
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
