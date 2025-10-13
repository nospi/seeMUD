#!/bin/bash

# SeeMUD Visual Client Launch Script
# This script sets up the proper PATH and launches Wails in development mode

echo "🎮 Launching SeeMUD Visual Client..."
echo "Setting up Go PATH..."

export PATH=$PATH:$(go env GOPATH)/bin

echo "Starting Wails development server..."
wails dev