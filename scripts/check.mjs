import { run, finished } from './process.mjs';
try {
  await finished(run('go', ['-C', 'backend', 'test', './...']));
  await finished(run('npm', ['run', 'typecheck']));
  await finished(run('npm', ['run', 'build']));
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
}
