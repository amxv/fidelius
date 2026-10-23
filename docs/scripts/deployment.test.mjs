import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const docs = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const root = path.dirname(docs);

test('site builds run only for documentation, either installer, or deployment configuration', () => {
  const workflow = readFileSync(path.join(root, '.github/workflows/docs-deploy.yml'), 'utf8');
  const relevant = [
    'docs/**', 'scripts/install.sh', 'scripts/install.ps1',
    'vercel.json', '.github/workflows/docs-deploy.yml'
  ];
  for (const event of ['push', 'pull_request']) {
    const block = workflow.match(new RegExp(`^  ${event}:\\n([\\s\\S]*?)(?=^  (?:push|pull_request|workflow_dispatch):|^permissions:)`, 'm'))?.[1];
    assert.ok(block, `${event} trigger missing`);
    assert.match(block, /branches: \[main\]/);
    assert.deepEqual([...block.matchAll(/^      - ['"]([^'"]+)['"]$/gm)].map(m => m[1]), relevant);
  }
  assert.match(workflow, /^  workflow_dispatch:$/m);
  assert.match(workflow, /github\.ref == 'refs\/heads\/main'/);
  assert.match(workflow, /secrets\.VERCEL_TOKEN/);
  assert.match(workflow, /vars\.VERCEL_ORG_ID/);
  assert.match(workflow, /vars\.VERCEL_PROJECT_ID/);
  assert.match(workflow, /cmp \.\.\/scripts\/install\.sh dist\/install\.sh/);
  assert.match(workflow, /cmp \.\.\/scripts\/install\.ps1 dist\/install\.ps1/);
  assert.match(workflow, /mv docs\/\.vercel\/output \.vercel\/output/);
  assert.match(workflow, /vercel@latest deploy --prebuilt --prod/);
});

test('the website serves both canonical root installers without modifying them', () => {
  for (const installer of ['install.sh', 'install.ps1']) {
    const source = readFileSync(path.join(docs, 'src/pages', `${installer}.ts`), 'utf8');
    assert.match(source, new RegExp(`readFile\\(join\\(process\\.cwd\\(\\), "\\.\\.", "scripts", "${installer.replace('.', '\\.')}"\\), "utf8"\\)`));
    assert.match(source, /return new Response\(script,/);
  }
});

test('Vercel accepts only Actions deployments; no quota-consuming ignored builds', () => {
  const config = JSON.parse(readFileSync(path.join(root, 'vercel.json'), 'utf8'));
  assert.equal(config.git.deploymentEnabled, false);
  assert.equal(Object.hasOwn(config, 'ignoreCommand'), false);
  assert.equal(config.outputDirectory, 'docs/dist');
});
