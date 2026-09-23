import { readFile, writeFile, mkdir, access } from 'node:fs/promises';
import { join, resolve } from 'node:path';
import { randomBytes } from 'node:crypto';
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
  try {
    await access(join(root, '.env'));
  } catch {
    const template = await readFile(join(root, '.env.example'), 'utf8');
    await writeFile(
      join(root, '.env'),
      template
        .replace(
          'DEMO_EMPLOYEE_PASSWORD=',
          `DEMO_EMPLOYEE_PASSWORD=${randomBytes(8).toString('hex')}`,
        )
        .replace('DEMO_HR_PASSWORD=', `DEMO_HR_PASSWORD=${randomBytes(8).toString('hex')}`),
      { mode: 0o600 },
    );
  }
  await finished(
    run('go', ['run', './backend/cmd/server'], {
      env: {
        ...process.env,
        VALIDATE_ONLY: '1',
        STATE_PATH: join(root, 'data/runtime/.setup-validation.json'),
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
    '\nSetup complete. Demo passwords are in .env (npm run credentials). Run npm run demo.',
  );
  console.log(
    'Existing runtime progress was preserved. .env and all dataset files are excluded from Git.',
  );
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
