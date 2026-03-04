export default {
  '*.{ts,tsx}': ['eslint --fix', 'prettier --write'],
  '*.{js,jsx,mjs,cjs}': ['prettier --write'],
  '*.{json,css,html,yaml,yml,md}': ['prettier --write'],
  'apps/api/**/*.go': () => 'cd apps/api && golangci-lint run --new-from-rev=HEAD ./...',
};
