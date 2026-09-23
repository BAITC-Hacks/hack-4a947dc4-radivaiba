import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { root } from './process.mjs';
try {
  const args = process.argv.slice(2);
  const index = args.indexOf('--login');
  if (index !== -1 && !args[index + 1]) throw new Error('Use --login E0001.');
  const fileIndex = args.indexOf('--file');
  const file =
    fileIndex === -1 ? join(root, 'data/credentials/accounts.json') : args[fileIndex + 1];
  if (!file) throw new Error('Use --file <private account export>.');
  const data = JSON.parse(await readFile(file, 'utf8'));
  const selected = data.accounts.filter((account) =>
    index !== -1
      ? account.login === args[index + 1]
      : account.login === 'E0001' || account.login === 'hr',
  );
  if (!selected.length)
    throw new Error(
      'Requested credentials are not in this private export. Existing passwords are never recovered from their hashes.',
    );
  for (const account of selected)
    console.log(`Login: ${account.login}\nPassword: ${account.password}\nRole: ${account.role}\n`);
} catch (error) {
  console.error(
    error.code === 'ENOENT'
      ? 'Provision accounts first: npm run accounts:provision. For Docker, copy the private export from /app/data/credentials/accounts.json.'
      : error.message,
  );
  process.exitCode = 1;
}
