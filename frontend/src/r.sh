#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 React App Refactoring Tool${NC}"
echo "================================"

# Check if App.jsx exists
if [ ! -f "App.jsx" ]; then
    echo -e "${RED}❌ Error: App.jsx not found in current directory${NC}"
    echo "Please run this script in the directory containing App.jsx"
    exit 1
fi

# Backup original file
echo -e "${YELLOW}📦 Creating backup...${NC}"
cp App.jsx App.jsx.backup
echo -e "${GREEN}✅ Backup created: App.jsx.backup${NC}"

# Check if src directory already exists
if [ -d "src" ]; then
    echo -e "${YELLOW}⚠️  Warning: 'src' directory already exists${NC}"
    read -p "Do you want to proceed and overwrite? (y/n): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${RED}❌ Refactoring cancelled${NC}"
        exit 1
    fi
    echo -e "${YELLOW}🗑️  Removing existing src directory...${NC}"
    rm -rf src
fi

# Create the refactor script if it doesn't exist
if [ ! -f "refactor.mjs" ]; then
    echo -e "${YELLOW}📝 Creating refactor.js script...${NC}"
    # You would paste the refactor.js content here or download it
    echo -e "${RED}❌ refactor.js not found. Please create it first.${NC}"
    exit 1
fi

# Run the refactoring
echo -e "${GREEN}🔧 Running refactoring...${NC}"
echo "================================"
node refactor.mjs

# Check if refactoring was successful
if [ $? -eq 0 ]; then
    echo "================================"
    echo -e "${GREEN}✨ Refactoring completed successfully!${NC}"
    echo
    echo -e "${GREEN}📁 New structure created in 'src' directory${NC}"
    echo
    echo "Next steps:"
    echo "1. Review the refactored code in the 'src' directory"
    echo "2. Update your imports to use: import App from './src/App'"
    echo "3. Test your application"
    echo "4. Once verified, you can remove App.jsx.backup"
    
    # Optional: Show the new structure
    echo
    echo -e "${YELLOW}📂 New file structure:${NC}"
    tree src -I 'node_modules' 2>/dev/null || find src -type f -name "*.jsx" -o -name "*.js" | sort
else
    echo -e "${RED}❌ Refactoring failed. Check the error messages above.${NC}"
    echo -e "${YELLOW}💡 Your original file is safe in App.jsx.backup${NC}"
    exit 1
fi
