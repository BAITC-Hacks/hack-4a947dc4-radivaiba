import { join } from 'node:path';
import { root, run, finished } from './process.mjs';

try {
  try {
    process.loadEnvFile(join(root, '.env'));
  } catch (error) {
    if (error.code !== 'ENOENT') throw error;
  }
  const [command, ...args] = process.argv.slice(2);
  if (!['migrate-legacy', 'provision-accounts'].includes(command))
    throw new Error('Unknown administration command.');
  await finished(run('go', ['run', `./backend/cmd/${command}`, ...args]));
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
