#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔄 Integration Script - Replacing Current Directory${NC}"
echo "===================================================="

# Check if src directory exists (from refactoring)
if [ ! -d "src" ]; then
    echo -e "${RED}❌ Error: 'src' directory not found!${NC}"
    echo "Please run the refactor script first."
    exit 1
fi

# Check if the refactored App.jsx exists
if [ ! -f "src/App.jsx" ]; then
    echo -e "${RED}❌ Error: 'src/App.jsx' not found!${NC}"
    echo "The refactoring might have failed."
    exit 1
fi

echo -e "${YELLOW}📋 This script will:${NC}"
echo "1. Move all refactored files to your current directory"
echo "2. Update your main entry point"
echo "3. Archive the old App.jsx"
echo ""
read -p "Continue? (y/n): " -n 1 -r
echo

if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}❌ Integration cancelled${NC}"
    exit 1
fi

# Step 1: Create backup directory
echo -e "\n${GREEN}📦 Step 1: Creating backup...${NC}"
BACKUP_DIR="backup_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

# Backup the original App.jsx if it exists
if [ -f "App.jsx" ]; then
    cp App.jsx "$BACKUP_DIR/App.jsx.original"
    echo -e "  ✅ Backed up App.jsx to $BACKUP_DIR/"
fi

# Backup any existing components directory
if [ -d "components" ]; then
    cp -r components "$BACKUP_DIR/components_original"
    echo -e "  ✅ Backed up existing components to $BACKUP_DIR/"
fi

# Backup any existing utils directory
if [ -d "utils" ]; then
    cp -r utils "$BACKUP_DIR/utils_original"
    echo -e "  ✅ Backed up existing utils to $BACKUP_DIR/"
fi

# Step 2: Move refactored files to current directory
echo -e "\n${GREEN}📁 Step 2: Moving refactored files...${NC}"

# Move components
if [ -d "src/components" ]; then
    rm -rf components 2>/dev/null
    mv src/components components
    echo -e "  ✅ Moved components directory"
fi

# Move utils
if [ -d "src/utils" ]; then
    rm -rf utils 2>/dev/null
    mv src/utils utils
    echo -e "  ✅ Moved utils directory"
fi

# Move the new App.jsx
if [ -f "src/App.jsx" ]; then
    rm -f App.jsx 2>/dev/null
    mv src/App.jsx App.jsx
    echo -e "  ✅ Replaced App.jsx with refactored version"
fi

# Move index.js if needed
if [ -f "src/index.js" ] && [ ! -f "index.js" ]; then
    mv src/index.js index.js
    echo -e "  ✅ Moved index.js"
fi

# Step 3: Update imports in your main entry file
echo -e "\n${GREEN}📝 Step 3: Checking main entry point...${NC}"

# Common entry points to check
ENTRY_FILES=("main.jsx" "index.jsx" "main.js" "index.js" "../index.html")

for file in "${ENTRY_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo -e "  📄 Found entry file: $file"
        
        # Check if it imports from the old location
        if grep -q "from './src/App" "$file"; then
            # Update the import
            sed -i.bak "s|from './src/App'|from './App'|g" "$file"
            echo -e "  ✅ Updated import in $file"
        elif grep -q "from './App'" "$file"; then
            echo -e "  ✅ Import already correct in $file"
        else
            echo -e "  ⚠️  Please check imports in $file manually"
        fi
    fi
done

# Step 4: Clean up
echo -e "\n${GREEN}🧹 Step 4: Cleaning up...${NC}"

# Remove the now-empty src directory
if [ -d "src" ]; then
    # Check if src is empty or only has package.json
    if [ -z "$(ls -A src | grep -v package.json)" ]; then
        rm -rf src
        echo -e "  ✅ Removed empty src directory"
    else
        echo -e "  ⚠️  src directory not empty, keeping it"
    fi
fi

# Step 5: Create a summary file
echo -e "\n${GREEN}📄 Step 5: Creating summary...${NC}"

cat > REFACTOR_SUMMARY.md << EOF
# Refactoring Summary

## Date: $(date)

### Files Created:
- \`App.jsx\` - Main application component
- \`components/\` - All UI components organized by type
  - \`sidebar/\` - Sidebar components
  - \`mobile/\` - Mobile-specific components
  - \`player/\` - Audio player components
  - \`drawers/\` - Drawer components
  - \`dialogs/\` - Dialog components
- \`utils/\` - Utility functions
  - \`streamMeta.js\` - Stream metadata helpers
  - \`formatters.js\` - Formatting utilities

### Backup Location:
- \`$BACKUP_DIR/\` - Contains all original files

### New Structure:
\`\`\`
.
├── App.jsx
├── components/
│   ├── sidebar/
│   │   └── StreamsList.jsx
│   ├── mobile/
│   │   └── StreamsDrawer.jsx
│   ├── player/
│   │   ├── PlayPauseButton.jsx
│   │   ├── EqualizerBars.jsx
│   │   ├── NowPlaying.jsx
│   │   └── TrackProgress.jsx
│   ├── drawers/
│   │   └── ArtistDrawer.jsx
│   └── dialogs/
│       └── InfoDialog.jsx
└── utils/
    ├── streamMeta.js
    └── formatters.js
\`\`\`

### Next Steps:
1. Test your application to ensure everything works
2. Review the component organization
3. Once confirmed working, you can delete the backup directory
EOF

echo -e "  ✅ Created REFACTOR_SUMMARY.md"

# Final summary
echo -e "\n${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "${GREEN}✨ Integration Complete!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════${NC}"
echo -e "\n${BLUE}📊 Summary:${NC}"
echo -e "  • Refactored code moved to current directory"
echo -e "  • Original files backed up to: ${YELLOW}$BACKUP_DIR/${NC}"
echo -e "  • New structure active in current directory"
echo -e "\n${YELLOW}⚠️  Important:${NC}"
echo -e "  1. Test your application now"
echo -e "  2. Check that all imports are working"
echo -e "  3. Once verified, you can delete: $BACKUP_DIR/"
echo -e "\n${GREEN}🎉 Your app is now cleanly refactored!${NC}"

# Show the new structure
echo -e "\n${BLUE}📁 New file structure:${NC}"
tree -I 'node_modules|backup_*' -L 3 2>/dev/null || {
    echo "App.jsx"
    find components utils -type f -name "*.jsx" -o -name "*.js" 2>/dev/null | sort | sed 's/^/  /'
}
