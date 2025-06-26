#!/bin/bash

# Test script to validate sandbox API tests for CI/CD integration
# This script mimics what will run in GitHub Actions

set -e

echo "🧪 Testing Sandbox API CI/CD Integration"
echo "========================================"

# Navigate to source directory  
cd "$(dirname "$0")/../src"

echo "1. Running Go unit tests for sandbox package..."
go test -v ./worker/sandbox

echo ""
echo "2. Running Go tests with race detection and coverage..."
go test -v -race -coverprofile=coverage.out ./worker/sandbox

echo ""
echo "3. Generating coverage report..."
go tool cover -func=coverage.out | grep "total:"

echo ""
echo "4. Generating HTML coverage report..."
go tool cover -html=coverage.out -o coverage.html

echo ""
echo "✅ All sandbox tests passed successfully!"
echo "📊 Coverage report generated: coverage.html"
echo "📝 Coverage data: coverage.out"

# Display key coverage metrics
echo ""
echo "📈 Key Coverage Metrics:"
echo "------------------------"
go tool cover -func=coverage.out | grep -E "(api\.go|safeSandbox\.go|sandbox\.go)" | grep -v "0.0%"

echo ""
echo "🎉 Sandbox API tests are ready for CI/CD integration!"