import { join } from 'node:path';
import { root, run, finished } from './process.mjs';

try {
  try {
    process.loadEnvFile(join(root, '.env'));
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  const databaseURL = process.env.TEST_DATABASE_URL || process.env.DATABASE_URL;
  if (!databaseURL) throw new Error('Set TEST_DATABASE_URL or run npm run setup first.');
  await finished(
    run('go', ['-C', 'backend', 'test', './internal/store', '-run', 'Postgres', '-v'], {
      env: { ...process.env, TEST_DATABASE_URL: databaseURL },
    }),
  );
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
