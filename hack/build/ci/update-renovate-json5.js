const fs = require('fs');
const JSON5 = require('json5');

const versionFile = process.argv[2];
const renovateFile = process.argv[3];

// Read test and renovate files
const releaseVersions = fs.readFileSync(versionFile, 'utf8').trim().split('\n');
const renovateConfig = JSON5.parse(fs.readFileSync(renovateFile, 'utf8'));

// Create version array
let baseVersions = ["\$default"]
baseVersions = baseVersions.concat(releaseVersions.map(v => v.replace('origin/', '')));

// Update baseBranches with versions from the version file
renovateConfig.baseBranches = baseVersions
renovateConfig.packageRules[0].matchBaseBranches = baseVersions

// Print updated renovate config
console.log(JSON5.stringify(renovateConfig, null, 2));
