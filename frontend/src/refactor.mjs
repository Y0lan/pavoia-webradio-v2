#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { dirname } from 'path';

// Get current directory for ES modules
const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);

// Configuration
const SOURCE_FILE = 'App.jsx';
const OUTPUT_DIR = 'src';

// Ensure output directory exists
function ensureDir(dir) {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
}

// Read the source file
function readSourceFile() {
  if (!fs.existsSync(SOURCE_FILE)) {
    console.error(`❌ Source file '${SOURCE_FILE}' not found!`);
    process.exit(1);
  }
  return fs.readFileSync(SOURCE_FILE, 'utf8');
}

// Component definitions with their extraction patterns
const componentStructure = {
  // Main component
  'App.jsx': {
    pattern: /export default function RadioApp\(\)[\s\S]*?^}\n\n/m,
    imports: [
      'import React, { useEffect, useMemo, useRef, useState } from "react";',
      'import { StreamsList } from "./components/sidebar/StreamsList";',
      'import { StreamsDrawer } from "./components/mobile/StreamsDrawer";',
      'import { PlayPauseButton } from "./components/player/PlayPauseButton";',
      'import { NowPlaying } from "./components/player/NowPlaying";',
      'import { ArtistDrawer } from "./components/drawers/ArtistDrawer";',
      'import { InfoDialog } from "./components/dialogs/InfoDialog";',
      'import { getMetaFor } from "./utils/streamMeta";',
      'import { fmtTime } from "./utils/formatters";'
    ],
    keepOriginalImports: true
  },

  // Sidebar components
  'components/sidebar/StreamsList.jsx': {
    pattern: /function StreamsList\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React from "react";',
      'import { getMetaFor } from "../../utils/streamMeta";'
    ],
    exportType: 'named'
  },

  // Mobile components
  'components/mobile/StreamsDrawer.jsx': {
    pattern: /function StreamsDrawer\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React from "react";',
      'import { StreamsList } from "../sidebar/StreamsList";'
    ],
    exportType: 'named'
  },

  // Player components
  'components/player/PlayPauseButton.jsx': {
    pattern: /function PlayPauseButton\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React from "react";',
      'import { EqualizerBars } from "./EqualizerBars";'
    ],
    exportType: 'named'
  },

  'components/player/EqualizerBars.jsx': {
    pattern: /function EqualizerBars\([^)]*\)[\s\S]*?^}\n/m,
    imports: ['import React from "react";'],
    exportType: 'named'
  },

  'components/player/NowPlaying.jsx': {
    pattern: /function NowPlaying\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React from "react";',
      'import { TrackProgress } from "./TrackProgress";'
    ],
    exportType: 'named'
  },

  'components/player/TrackProgress.jsx': {
    pattern: /function TrackProgress\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React from "react";',
      'import { fmtTime } from "../../utils/formatters";'
    ],
    exportType: 'named'
  },

  // Drawer components
  'components/drawers/ArtistDrawer.jsx': {
    pattern: /function ArtistDrawer\([^)]*\)[\s\S]*?^}\n/m,
    imports: [
      'import React, { useState, useEffect } from "react";'
    ],
    exportType: 'named'
  },

  // Dialog components
  'components/dialogs/InfoDialog.jsx': {
    pattern: /function InfoDialog\([^)]*\)[\s\S]*?^}\n/m,
    imports: ['import React from "react";'],
    exportType: 'named'
  },

  // Utility functions
  'utils/streamMeta.js': {
    pattern: /function getMetaFor\([^)]*\)[\s\S]*?^}\n/m,
    imports: [],
    exportType: 'named',
    isUtility: true
  },

  'utils/formatters.js': {
    pattern: /function fmtTime\([^)]*\)[\s\S]*?^}\n/m,
    imports: [],
    exportType: 'named',
    isUtility: true
  }
};

// Extract component code
function extractComponent(source, pattern) {
  const match = source.match(pattern);
  return match ? match[0] : null;
}

// Process main App component
function processMainApp(source) {
  // Extract everything up to the first sub-component definition
  const lines = source.split('\n');
  const componentStartIndex = lines.findIndex(line => 
    line.includes('function StreamsList') || 
    line.includes('function getMetaFor')
  );
  
  if (componentStartIndex === -1) {
    return source;
  }

  // Get main component code
  let mainCode = lines.slice(0, componentStartIndex).join('\n');
  
  // Clean up the ending
  mainCode = mainCode.replace(/}\s*$/, '');
  
  // Find the last actual code line (not just whitespace or comments)
  const codeLines = mainCode.split('\n');
  let lastCodeIndex = codeLines.length - 1;
  while (lastCodeIndex >= 0 && codeLines[lastCodeIndex].trim() === '') {
    lastCodeIndex--;
  }
  
  return codeLines.slice(0, lastCodeIndex + 1).join('\n') + '\n}';
}

// Generate file content
function generateFileContent(componentName, config, source) {
  let content = '';
  let componentCode = '';

  if (componentName === 'App.jsx') {
    componentCode = processMainApp(source);
  } else {
    componentCode = extractComponent(source, config.pattern);
  }

  if (!componentCode) {
    console.warn(`⚠️  Could not extract component: ${componentName}`);
    return null;
  }

  // Add imports
  content += config.imports.join('\n') + '\n\n';

  // Add component code with proper export
  if (config.exportType === 'named') {
    content += 'export ' + componentCode;
  } else if (componentName === 'App.jsx') {
    // For main App, keep as default export
    content += componentCode.replace('export default function RadioApp', 'export default function App');
  } else {
    content += componentCode;
  }

  return content;
}

// Main refactoring function
function refactorApp() {
  console.log('🚀 Starting refactoring process...\n');

  const source = readSourceFile();
  console.log(`📖 Read ${SOURCE_FILE} (${source.length} characters)\n`);

  // Create directory structure
  const dirs = [
    OUTPUT_DIR,
    path.join(OUTPUT_DIR, 'components'),
    path.join(OUTPUT_DIR, 'components', 'sidebar'),
    path.join(OUTPUT_DIR, 'components', 'mobile'),
    path.join(OUTPUT_DIR, 'components', 'player'),
    path.join(OUTPUT_DIR, 'components', 'drawers'),
    path.join(OUTPUT_DIR, 'components', 'dialogs'),
    path.join(OUTPUT_DIR, 'utils')
  ];

  dirs.forEach(dir => {
    ensureDir(dir);
    console.log(`📁 Created directory: ${dir}`);
  });

  console.log('\n📝 Extracting and writing components...\n');

  // Process each component
  const results = {
    success: [],
    failed: []
  };

  Object.entries(componentStructure).forEach(([fileName, config]) => {
    const filePath = path.join(OUTPUT_DIR, fileName);
    const content = generateFileContent(fileName, config, source);

    if (content) {
      fs.writeFileSync(filePath, content);
      results.success.push(fileName);
      console.log(`✅ Created: ${filePath}`);
    } else {
      results.failed.push(fileName);
      console.log(`❌ Failed: ${fileName}`);
    }
  });

  // Create index.js for easier imports
  const indexContent = `// Main App export
export { default } from './App';

// Component exports
export * from './components/sidebar/StreamsList';
export * from './components/mobile/StreamsDrawer';
export * from './components/player/PlayPauseButton';
export * from './components/player/EqualizerBars';
export * from './components/player/NowPlaying';
export * from './components/player/TrackProgress';
export * from './components/drawers/ArtistDrawer';
export * from './components/dialogs/InfoDialog';

// Utility exports
export * from './utils/streamMeta';
export * from './utils/formatters';
`;

  fs.writeFileSync(path.join(OUTPUT_DIR, 'index.js'), indexContent);
  console.log(`\n✅ Created: ${path.join(OUTPUT_DIR, 'index.js')}`);

  // Summary
  console.log('\n' + '='.repeat(50));
  console.log('📊 Refactoring Summary:');
  console.log('='.repeat(50));
  console.log(`✅ Successfully created: ${results.success.length} files`);
  if (results.failed.length > 0) {
    console.log(`❌ Failed to create: ${results.failed.length} files`);
    console.log(`   Failed files: ${results.failed.join(', ')}`);
  }
  console.log('\n🎉 Refactoring complete!');
  console.log('\n📌 Next steps:');
  console.log('1. Review the generated files in the "src" directory');
  console.log('2. Update your main application to import from "src/App"');
  console.log('3. Test that everything works correctly');
  console.log('4. Delete the original App.jsx file once verified');
}

// Run the refactoring
try {
  refactorApp();
} catch (error) {
  console.error('\n❌ Error during refactoring:', error.message);
  process.exit(1);
}
