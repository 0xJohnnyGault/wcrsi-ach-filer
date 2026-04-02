package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xuri/excelize/v2"
)

type Config struct {
	SourceFolder string `json:"sourceFolder"`
	DestFolder   string `json:"destFolder"`
}

type LogEntry struct {
	Type    string `json:"type"` // "success" or "error"
	Message string `json:"message"`
}

type ProcessResult struct {
	Logs    []LogEntry `json:"logs"`
	Success bool       `json:"success"`
}

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".achfiler")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func (a *App) GetConfig() (*Config, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &Config{}, nil
	}
	return &cfg, nil
}

func (a *App) SaveConfig(cfg Config) error {
	p, err := configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

func (a *App) SelectFolder(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
	})
}

func (a *App) ProcessFiles(sourceFolder, destFolder string) ProcessResult {
	result := ProcessResult{Success: true}
	addLog := func(logType, msg string) {
		result.Logs = append(result.Logs, LogEntry{Type: logType, Message: msg})
		if logType == "error" {
			result.Success = false
		}
	}

	// Find first XLS/XLSX file in source folder
	xlsFile := ""
	entries, err := os.ReadDir(sourceFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read source folder: %v", err))
		return result
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".xls" || ext == ".xlsx" {
			xlsFile = filepath.Join(sourceFolder, e.Name())
			break
		}
	}
	if xlsFile == "" {
		addLog("error", "No XLS/XLSX file found in source folder")
		return result
	}
	addLog("success", fmt.Sprintf("Found spreadsheet: %s", filepath.Base(xlsFile)))

	// Open the spreadsheet
	f, err := excelize.OpenFile(xlsFile)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot open spreadsheet: %v", err))
		return result
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		addLog("error", "No sheets found in spreadsheet")
		return result
	}

	// Find "Account" column in Row 2
	rows, err := f.GetRows(sheetName)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read rows: %v", err))
		return result
	}
	if len(rows) < 2 {
		addLog("error", "Spreadsheet has fewer than 2 rows")
		return result
	}

	headerRow := rows[1] // Row 2 (0-indexed: 1)
	accountCol := -1
	for i, cell := range headerRow {
		if strings.EqualFold(strings.TrimSpace(cell), "account") {
			accountCol = i
			break
		}
	}
	if accountCol == -1 {
		addLog("error", "No 'Account' column found in Row 2")
		return result
	}

	// Read destination sub-directories
	destEntries, err := os.ReadDir(destFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read destination folder: %v", err))
		return result
	}
	var destDirs []string
	for _, e := range destEntries {
		if e.IsDir() {
			destDirs = append(destDirs, e.Name())
		}
	}

	// Collect source files to copy (all files in source folder)
	var sourceFiles []string
	for _, e := range entries {
		if !e.IsDir() {
			sourceFiles = append(sourceFiles, e.Name())
		}
	}

	// Track which accounts we've already processed to avoid duplicate copies
	processed := make(map[string]bool)

	// Loop through data rows (starting at Row 3, index 2)
	for rowIdx := 2; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if accountCol >= len(row) {
			continue
		}
		acctNum := strings.TrimSpace(row[accountCol])
		if acctNum == "" {
			continue
		}
		if processed[acctNum] {
			continue
		}
		processed[acctNum] = true

		// Search destination dirs for one containing the account number
		foundDir := ""
		for _, dir := range destDirs {
			if strings.Contains(dir, acctNum) {
				foundDir = dir
				break
			}
		}

		if foundDir == "" {
			addLog("error", fmt.Sprintf("Account %s: No matching directory found in destination", acctNum))
			continue
		}

		// Create Payments subdirectory
		paymentsDir := filepath.Join(destFolder, foundDir, "Payments")
		if err := os.MkdirAll(paymentsDir, 0755); err != nil {
			addLog("error", fmt.Sprintf("Account %s: Cannot create Payments folder: %v", acctNum, err))
			continue
		}

		// Copy all source files to Payments dir
		copyErr := false
		for _, fname := range sourceFiles {
			src := filepath.Join(sourceFolder, fname)
			dst := filepath.Join(paymentsDir, fname)
			if err := copyFile(src, dst); err != nil {
				addLog("error", fmt.Sprintf("Account %s: Failed to copy %s: %v", acctNum, fname, err))
				copyErr = true
			}
		}
		if !copyErr {
			addLog("success", fmt.Sprintf("Account %s: Copied %d files to %s/Payments", acctNum, len(sourceFiles), foundDir))
		}
	}

	if len(processed) == 0 {
		addLog("error", "No account numbers found in the spreadsheet")
	}

	return result
}

func (a *App) CheckFiler(sourceFolder, destFolder string) ProcessResult {
	result := ProcessResult{Success: true}
	addLog := func(logType, msg string) {
		result.Logs = append(result.Logs, LogEntry{Type: logType, Message: msg})
		if logType == "error" {
			result.Success = false
		}
	}

	// Read source directory for PDF files
	entries, err := os.ReadDir(sourceFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read source folder: %v", err))
		return result
	}

	var pdfFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(e.Name()), ".pdf") {
			pdfFiles = append(pdfFiles, e.Name())
		}
	}

	if len(pdfFiles) == 0 {
		addLog("error", "No PDF files found in source folder")
		return result
	}

	// Read destination sub-directories
	destEntries, err := os.ReadDir(destFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read destination folder: %v", err))
		return result
	}
	var destDirs []string
	for _, e := range destEntries {
		if e.IsDir() {
			destDirs = append(destDirs, e.Name())
		}
	}

	// Process each PDF file
	for _, pdfName := range pdfFiles {
		// Extract account number: last 10 characters of filename (before .pdf extension)
		baseName := strings.TrimSuffix(pdfName, filepath.Ext(pdfName))
		if len(baseName) < 10 {
			addLog("error", fmt.Sprintf("%s: Filename too short to extract 10-digit account number", pdfName))
			continue
		}
		acctNum := baseName[len(baseName)-10:]

		// Search destination dirs for one containing the account number
		foundDir := ""
		for _, dir := range destDirs {
			if strings.Contains(dir, acctNum) {
				foundDir = dir
				break
			}
		}

		if foundDir == "" {
			addLog("error", fmt.Sprintf("%s: No matching directory found for account %s", pdfName, acctNum))
			continue
		}

		// Create Payments subdirectory if needed
		paymentsDir := filepath.Join(destFolder, foundDir, "Payments")
		if err := os.MkdirAll(paymentsDir, 0755); err != nil {
			addLog("error", fmt.Sprintf("%s: Cannot create Payments folder: %v", pdfName, err))
			continue
		}

		// Copy PDF to Payments dir
		src := filepath.Join(sourceFolder, pdfName)
		dst := filepath.Join(paymentsDir, pdfName)
		if err := copyFile(src, dst); err != nil {
			addLog("error", fmt.Sprintf("%s: Failed to copy: %v", pdfName, err))
		} else {
			addLog("success", fmt.Sprintf("%s: Copied to %s/Payments (account %s)", pdfName, foundDir, acctNum))
		}
	}

	return result
}

func (a *App) CBPFiler(sourceFolder, destFolder string) ProcessResult {
	result := ProcessResult{Success: true}
	addLog := func(logType, msg string) {
		result.Logs = append(result.Logs, LogEntry{Type: logType, Message: msg})
		if logType == "error" {
			result.Success = false
		}
	}

	// Find first XLS/XLSX file in source folder
	xlsFile := ""
	entries, err := os.ReadDir(sourceFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read source folder: %v", err))
		return result
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".xls" || ext == ".xlsx" {
			xlsFile = filepath.Join(sourceFolder, e.Name())
			break
		}
	}
	if xlsFile == "" {
		addLog("error", "No XLS/XLSX file found in source folder")
		return result
	}
	addLog("success", fmt.Sprintf("Found spreadsheet: %s", filepath.Base(xlsFile)))

	// Open the spreadsheet
	f, err := excelize.OpenFile(xlsFile)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot open spreadsheet: %v", err))
		return result
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		addLog("error", "No sheets found in spreadsheet")
		return result
	}

	// Find "Reference/Invoice#" column in Row 1
	rows, err := f.GetRows(sheetName)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read rows: %v", err))
		return result
	}
	if len(rows) < 1 {
		addLog("error", "Spreadsheet is empty")
		return result
	}

	headerRow := rows[0] // Row 1 (0-indexed: 0)
	refCol := -1
	for i, cell := range headerRow {
		if strings.EqualFold(strings.TrimSpace(cell), "reference/invoice#") {
			refCol = i
			break
		}
	}
	if refCol == -1 {
		addLog("error", "No 'Reference/Invoice#' column found in Row 1")
		return result
	}

	// Read destination sub-directories
	destEntries, err := os.ReadDir(destFolder)
	if err != nil {
		addLog("error", fmt.Sprintf("Cannot read destination folder: %v", err))
		return result
	}
	var destDirs []string
	for _, e := range destEntries {
		if e.IsDir() {
			destDirs = append(destDirs, e.Name())
		}
	}

	// Collect source files to copy (all files in source folder)
	var sourceFiles []string
	for _, e := range entries {
		if !e.IsDir() {
			sourceFiles = append(sourceFiles, e.Name())
		}
	}

	// Track which references we've already processed to avoid duplicate copies
	processed := make(map[string]bool)

	// Loop through data rows (starting at Row 2, index 1)
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		if refCol >= len(row) {
			continue
		}
		refNum := strings.TrimSpace(row[refCol])
		if refNum == "" {
			continue
		}
		if processed[refNum] {
			continue
		}
		processed[refNum] = true

		// Search destination dirs for one containing the reference/invoice number
		foundDir := ""
		for _, dir := range destDirs {
			if strings.Contains(dir, refNum) {
				foundDir = dir
				break
			}
		}

		if foundDir == "" {
			addLog("error", fmt.Sprintf("Reference %s: No matching directory found in destination", refNum))
			continue
		}

		// Create Payments subdirectory
		paymentsDir := filepath.Join(destFolder, foundDir, "Payments")
		if err := os.MkdirAll(paymentsDir, 0755); err != nil {
			addLog("error", fmt.Sprintf("Reference %s: Cannot create Payments folder: %v", refNum, err))
			continue
		}

		// Copy all source files to Payments dir
		copyErr := false
		for _, fname := range sourceFiles {
			src := filepath.Join(sourceFolder, fname)
			dst := filepath.Join(paymentsDir, fname)
			if err := copyFile(src, dst); err != nil {
				addLog("error", fmt.Sprintf("Reference %s: Failed to copy %s: %v", refNum, fname, err))
				copyErr = true
			}
		}
		if !copyErr {
			addLog("success", fmt.Sprintf("Reference %s: Copied %d files to %s/Payments", refNum, len(sourceFiles), foundDir))
		}
	}

	if len(processed) == 0 {
		addLog("error", "No reference/invoice numbers found in the spreadsheet")
	}

	return result
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
