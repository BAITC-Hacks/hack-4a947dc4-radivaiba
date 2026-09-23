import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { root } from './process.mjs';
try {
  const text = await readFile(join(root, '.env'), 'utf8');
  for (const line of text.split(/\r?\n/))
    if (/^DEMO_(EMPLOYEE_ID|EMPLOYEE_PASSWORD|HR_PASSWORD)=/.test(line)) console.log(line);
} catch {
  console.error('Run npm run setup first.');
  process.exitCode = 1;
}
