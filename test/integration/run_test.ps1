$ErrorActionPreference = "Stop"

Write-Host "Starting Docker Integration Tests..." -ForegroundColor Cyan

# 1. Start Postgres Containers
Write-Host "Spinning up Docker containers..." -ForegroundColor Green
docker compose up -d

Write-Host "Waiting for source-db to be ready..."
$sourceReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $result = docker exec integration-source-db-1 pg_isready -U testuser -d sourcedb 2>$null
    if ($result -match "accepting connections") {
        $sourceReady = $true
        break
    }
    Start-Sleep -Seconds 1
}

Write-Host "Waiting for target-db to be ready..."
$targetReady = $false
for ($i = 0; $i -lt 30; $i++) {
    $result = docker exec integration-target-db-1 pg_isready -U testuser -d targetdb 2>$null
    if ($result -match "accepting connections") {
        $targetReady = $true
        break
    }
    Start-Sleep -Seconds 1
}

if (-not $sourceReady -or -not $targetReady) {
    Write-Error "Databases failed to initialize in time."
    docker compose down -v
    exit 1
}

Write-Host "Databases are ready! Running pg-diff..." -ForegroundColor Green

# Wait an extra 2 seconds for internal Postgres catalog syncs if any
Start-Sleep -Seconds 2

# 2. Build and Run Current pg-diff CLI
cd ../../
go build -o pg-diff.exe ./cmd/pg-diff

$sourceUrl = "postgres://testuser:testpassword@localhost:15432/sourcedb?sslmode=disable"
$targetUrl = "postgres://testuser:testpassword@localhost:15433/targetdb?sslmode=disable"

# Execute Diff
$outputFileName = "test_output.sql"
./pg-diff.exe --source $sourceUrl --target $targetUrl > test\integration\$outputFileName

cd test/integration

# 3. Analyze output mapping
$diffContent = Get-Content -Raw $outputFileName

Write-Host "Diff Generation complete. Verifying output..." -ForegroundColor Yellow

$passed = $true

function Validate-Output {
    param([string]$pattern, [string]$desc)
    if ($diffContent -match $pattern) {
        Write-Host "[PASS] Found: $desc" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] Missing: $desc" -ForegroundColor Red
        $script:passed = $false
    }
}

# Source -> Target Validation Rules (Upgrading SOURCE to match TARGET)
Validate-Output "(?i)DROP TYPE IF EXISTS user_status" "Drops Source-only Enum Type"
Validate-Output "(?i)DROP SEQUENCE IF EXISTS dropping_seq" "Drops Source-only Sequence"
Validate-Output "(?i)DROP TABLE old_legacy_table" "Drops Source-only Table"
Validate-Output "(?i)DROP EXTENSION IF EXISTS uuid-ossp" "Drops Source-only extension"
Validate-Output "(?i)DROP VIEW IF EXISTS active_users" "Drops Source-only View"
Validate-Output "(?i)DROP FUNCTION IF EXISTS get_legacy_data" "Drops Source-only Function"

Validate-Output "(?i)CREATE EXTENSION IF NOT EXISTS citext VERSION '1\.\d+'" "Creates Target-only extension"
Validate-Output "(?i)CREATE TYPE extra_status" "Creates Target-only Type"
Validate-Output "(?i)CREATE SEQUENCE target_only_seq" "Creates Target-only Sequence"
Validate-Output "(?i)CREATE TABLE extra_target_table" "Creates Target-only Table"
Validate-Output "(?i)CREATE OR REPLACE FUNCTION get_target_only_data" "Creates Target-only Function"

# Alterations (Privileges and Columns)
Validate-Output "(?i)ALTER TABLE users ADD COLUMN last_login" "Adds Target-only column"
Validate-Output "(?i)ALTER TABLE users DROP COLUMN status" "Drops Source-only column"
Validate-Output "(?i)COMMENT ON TABLE users IS 'Modernized user table'" "Updates Table Comment"
Validate-Output "(?i)COMMENT ON COLUMN users\.id IS 'Primary identifier - modernized'" "Updates Column Comment"
Validate-Output "(?i)COMMENT ON COLUMN users\.email IS NULL" "Adds Missing Column Comment"
Validate-Output "(?i)GRANT DELETE ON TABLE users TO read_only_bob" "Grants missing privilege from Target"


if ($passed) {
    Write-Host "ALL INTEGRATION TESTS PASSED!" -ForegroundColor Green
} else {
    Write-Host "SOME INTEGRATION TESTS FAILED. See log output: $outputFileName" -ForegroundColor Red
}

# 4. Clean up
Write-Host "Tearing down docker containers..." -ForegroundColor Cyan
docker compose down -v

if (-not $passed) {
    exit 1
}
