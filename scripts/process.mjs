import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';
export const root = fileURLToPath(new URL('../', import.meta.url));
export function run(command, args, options = {}) {
  // npm.cmd needs cmd.exe on Windows; arguments here are repository-owned constants.
  const child =
    process.platform === 'win32' && command === 'npm'
      ? spawn(process.env.ComSpec || 'cmd.exe', ['/d', '/s', '/c', `npm ${args.join(' ')}`], {
          cwd: root,
          stdio: 'inherit',
          windowsHide: true,
          ...options,
        })
      : spawn(command, args, { cwd: root, stdio: 'inherit', windowsHide: true, ...options });
  return child;
}
export function finished(child) {
  return new Promise((resolve, reject) => {
    child.once('error', reject);
    child.once('exit', (code) =>
      code === 0 ? resolve() : reject(new Error(`Command exited with code ${code}`)),
    );
  });
}
