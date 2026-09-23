import { root, run, finished } from './process.mjs';
import { join } from 'node:path';
try {
  process.loadEnvFile(join(root, '.env'));
} catch (error) {
  if (error.code !== 'ENOENT') throw error;
}
const mode = process.argv[2] || 'demo';
const children = new Set();
let stopping = false;
function start(command, args, env = process.env) {
  const child = run(command, args, { env });
  children.add(child);
  child.once('exit', () => children.delete(child));
  return child;
}
function stop() {
  if (stopping) return;
  stopping = true;
  for (const child of children) {
    if (process.platform === 'win32' && child.pid)
      run('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore' });
    else child.kill('SIGTERM');
  }
}
process.once('SIGINT', stop);
process.once('SIGTERM', stop);
try {
  if (mode === 'demo') {
    await finished(start('npm', ['run', 'build']));
    await finished(start('go', ['run', './backend/cmd/server']));
  } else if (mode === 'dev') {
    const port = process.env.PORT || '8080';
    const api = start('go', ['run', './backend/cmd/server'], {
      ...process.env,
      DEV_ORIGIN: 'http://localhost:5173',
    });
    const web = start('npm', ['run', 'dev', '--workspace', 'frontend'], {
      ...process.env,
      API_PROXY_TARGET: `http://127.0.0.1:${port}`,
    });
    await Promise.race([finished(api), finished(web)]);
    stop();
  } else throw new Error(`Unknown mode: ${mode}`);
} catch (error) {
  if (!stopping) {
    console.error(error.message);
    process.exitCode = 1;
  }
  stop();
}
