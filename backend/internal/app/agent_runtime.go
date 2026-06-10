package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func agentTaskBranchName(now time.Time) string {
	return "agent/frontend-from-api-" + now.Format("20060102-150405")
}

func createAgentTaskBranch(workspaceRoot string, branchName string) error {
	if strings.TrimSpace(branchName) == "" || strings.Contains(branchName, "..") {
		return fmt.Errorf("invalid branch name")
	}
	if _, err := runGit(workspaceRoot, "checkout", "-b", branchName); err != nil {
		return err
	}

	return nil
}

func checkoutAgentTaskBranch(workspaceRoot string, branchName string) error {
	if strings.TrimSpace(branchName) == "" || strings.Contains(branchName, "..") {
		return fmt.Errorf("invalid branch name")
	}
	if _, err := runGit(workspaceRoot, "checkout", branchName); err != nil {
		return err
	}

	return nil
}

func safeWorkspacePath(workspaceRoot string, relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("path must be relative")
	}

	root := filepath.Clean(workspaceRoot)
	absPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(relPath)))
	relative, err := filepath.Rel(root, absPath)
	if err != nil {
		return "", fmt.Errorf("check workspace path: %w", err)
	}
	if relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || relative == ".." {
		return "", fmt.Errorf("path escapes workspace")
	}

	return absPath, nil
}

func normalizeAgentAction(action AgentAction) AgentAction {
	action.Type = strings.TrimSpace(action.Type)
	action.Path = strings.TrimSpace(action.Path)
	action.Query = strings.TrimSpace(action.Query)
	action.Command = strings.TrimSpace(action.Command)
	action.Reason = strings.TrimSpace(action.Reason)
	return action
}

func executeAgentAction(workspaceRoot string, validationCommands []string, action AgentAction) AgentObservation {
	switch action.Type {
	case "read_file":
		return executeReadFileAction(workspaceRoot, action.Path)
	case "list_dir":
		return executeListDirAction(workspaceRoot, action.Path)
	case "search_text":
		return executeSearchTextAction(workspaceRoot, action.Query)
	case "write_file":
		return executeWriteFileAction(workspaceRoot, action.Path, action.Content)
	case "run_command":
		return executeRunCommandAction(workspaceRoot, validationCommands, action.Command)
	case "finish":
		return AgentObservation{Status: "ok", Message: nonEmptyString(action.Reason, "Agent finished.")}
	default:
		return AgentObservation{Status: "unsupported", Message: "Unsupported action type: " + action.Type}
	}
}

func executeReadFileAction(workspaceRoot string, relPath string) AgentObservation {
	absPath, err := safeWorkspacePath(workspaceRoot, relPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: err.Error(), Path: relPath}
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: "read file stat failed: " + err.Error(), Path: relPath}
	}
	if info.IsDir() {
		return AgentObservation{Status: "failed", Message: "path is a directory", Path: relPath}
	}
	if info.Size() > 256*1024 {
		return AgentObservation{Status: "failed", Message: "file is too large to read safely", Path: relPath}
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: "read file failed: " + err.Error(), Path: relPath}
	}

	return AgentObservation{
		Status:  "ok",
		Message: fmt.Sprintf("Read %d bytes.", len(content)),
		Path:    filepath.ToSlash(filepath.Clean(relPath)),
		Content: truncateText(string(content), 12000),
	}
}

func executeListDirAction(workspaceRoot string, relPath string) AgentObservation {
	if strings.TrimSpace(relPath) == "" {
		relPath = "."
	}
	absPath, err := safeWorkspacePath(workspaceRoot, relPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: err.Error(), Path: relPath}
	}
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: "list dir failed: " + err.Error(), Path: relPath}
	}

	items := []string{}
	for _, entry := range entries {
		if shouldSkipWorkspaceDir(entry.Name()) {
			continue
		}
		itemPath := filepath.ToSlash(filepath.Join(relPath, entry.Name()))
		if relPath == "." {
			itemPath = entry.Name()
		}
		if entry.IsDir() {
			itemPath += "/"
		}
		items = append(items, itemPath)
		if len(items) >= 80 {
			items = append(items, "... truncated ...")
			break
		}
	}

	return AgentObservation{
		Status:  "ok",
		Message: fmt.Sprintf("Listed %d entries.", len(items)),
		Path:    filepath.ToSlash(filepath.Clean(relPath)),
		Items:   items,
	}
}

func executeSearchTextAction(workspaceRoot string, query string) AgentObservation {
	query = strings.TrimSpace(query)
	if query == "" {
		return AgentObservation{Status: "failed", Message: "search query is required"}
	}

	matches := []string{}
	err := filepath.WalkDir(workspaceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if path != workspaceRoot && shouldSkipWorkspaceDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if len(matches) >= 80 {
			return filepath.SkipAll
		}
		info, err := entry.Info()
		if err != nil || info.Size() > 256*1024 {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(content)
		if !strings.Contains(strings.ToLower(text), strings.ToLower(query)) {
			return nil
		}
		relPath, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			return nil
		}
		for lineNumber, line := range strings.Split(text, "\n") {
			if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(relPath), lineNumber+1, strings.TrimSpace(line)))
				if len(matches) >= 80 {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return AgentObservation{Status: "failed", Message: "search failed: " + err.Error()}
	}

	return AgentObservation{
		Status:  "ok",
		Message: fmt.Sprintf("Found %d matches.", len(matches)),
		Matches: matches,
	}
}

func executeWriteFileAction(workspaceRoot string, relPath string, content string) AgentObservation {
	if strings.TrimSpace(relPath) == "" {
		return AgentObservation{Status: "failed", Message: "write path is required"}
	}
	if len(content) > 512*1024 {
		return AgentObservation{Status: "failed", Message: "write content is too large", Path: relPath}
	}
	absPath, err := safeWorkspacePath(workspaceRoot, relPath)
	if err != nil {
		return AgentObservation{Status: "failed", Message: err.Error(), Path: relPath}
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return AgentObservation{Status: "failed", Message: "create parent directory failed: " + err.Error(), Path: relPath}
	}
	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		return AgentObservation{Status: "failed", Message: "write file failed: " + err.Error(), Path: relPath}
	}

	return AgentObservation{
		Status:  "ok",
		Message: fmt.Sprintf("Wrote %d bytes.", len(content)),
		Path:    filepath.ToSlash(filepath.Clean(relPath)),
	}
}

func executeRunCommandAction(workspaceRoot string, validationCommands []string, command string) AgentObservation {
	command = strings.TrimSpace(command)
	if command == "" {
		return AgentObservation{Status: "failed", Message: "command is required"}
	}
	if !containsExactString(validationCommands, command) {
		return AgentObservation{Status: "unsupported", Message: "command is not in discovered validation allowlist", Command: command}
	}
	verification := runValidationCommand(workspaceRoot, command)
	return AgentObservation{
		Status:  verificationStatusToObservationStatus(verification.Status),
		Message: "Command " + verification.Status + ".",
		Command: command,
		Output:  verification.Output,
	}
}

func verificationStatusToObservationStatus(status string) string {
	if status == "passed" {
		return "ok"
	}
	return status
}

func summarizeAgentObservation(action AgentAction, observation AgentObservation) string {
	target := action.Path
	if target == "" {
		target = action.Query
	}
	if target == "" {
		target = action.Command
	}
	if target == "" {
		return observation.Message
	}
	return fmt.Sprintf("%s %s: %s", action.Type, target, observation.Message)
}

func upsertAgentReadFile(files []AgentReadFile, file AgentReadFile) []AgentReadFile {
	for index := range files {
		if files[index].Path == file.Path {
			files[index] = file
			return files
		}
	}
	return append(files, file)
}

func lastAgentTaskSteps(steps []AgentTaskStep, limit int) []AgentTaskStep {
	if limit <= 0 || len(steps) <= limit {
		return cloneAgentTaskSteps(steps)
	}
	return cloneAgentTaskSteps(steps[len(steps)-limit:])
}

func appendUniqueString(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" || containsExactString(values, value) {
		return values
	}
	return append(values, value)
}

func containsExactString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func nonEmptyString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "\n... truncated ..."
}

func runWhitelistedValidationCommand(workspaceRoot string, commands []string) AgentVerification {
	if len(commands) == 0 {
		return AgentVerification{
			Status:  "failed",
			Command: "not detected",
			Output:  []string{"No validation command was discovered for this workspace."},
		}
	}

	for _, command := range commands {
		result := runValidationCommand(workspaceRoot, command)
		if result.Status != "unsupported" {
			return result
		}
	}

	return AgentVerification{
		Status:  "failed",
		Command: commands[0],
		Output:  []string{"No discovered validation command is supported by the safe executor."},
	}
}

func runValidationCommand(workspaceRoot string, command string) AgentVerification {
	workdir := workspaceRoot
	commandText := strings.TrimSpace(command)
	if commandText == "" {
		return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Empty command."}}
	}

	if strings.Contains(commandText, "&&") {
		parts := strings.Split(commandText, "&&")
		if len(parts) != 2 {
			return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Only one cd prefix is supported."}}
		}

		cdPart := strings.TrimSpace(parts[0])
		if !strings.HasPrefix(cdPart, "cd ") {
			return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Only cd prefixes are supported."}}
		}

		nextWorkdir, err := safeWorkspacePath(workspaceRoot, strings.TrimSpace(strings.TrimPrefix(cdPart, "cd ")))
		if err != nil {
			return AgentVerification{Status: "failed", Command: command, Output: []string{err.Error()}}
		}
		workdir = nextWorkdir
		commandText = strings.TrimSpace(parts[1])
	}

	name, args, ok := parseValidationCommand(commandText)
	if !ok {
		return AgentVerification{Status: "unsupported", Command: command, Output: []string{"Command is not in the safe executor allowlist."}}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	execCommand := exec.CommandContext(ctx, name, args...)
	execCommand.Dir = workdir
	output, err := execCommand.CombinedOutput()
	outputLines := commandOutputLines(output)
	if ctx.Err() == context.DeadlineExceeded {
		outputLines = append(outputLines, "Command timed out after 60 seconds.")
		return AgentVerification{Status: "failed", Command: command, Output: outputLines}
	}
	if err != nil {
		outputLines = append(outputLines, err.Error())
		return AgentVerification{Status: "failed", Command: command, Output: outputLines}
	}

	if len(outputLines) == 0 {
		outputLines = []string{"Command completed successfully."}
	}
	return AgentVerification{Status: "passed", Command: command, Output: outputLines}
}

func parseValidationCommand(command string) (string, []string, bool) {
	fields := splitCommandFields(command)
	if len(fields) == 0 {
		return "", nil, false
	}

	switch {
	case len(fields) == 3 && fields[0] == "npm" && fields[1] == "run":
		return "npm", fields[1:], true
	case len(fields) == 3 && fields[0] == "go" && fields[1] == "test" && fields[2] == "./...":
		return "go", fields[1:], true
	case len(fields) >= 3 && fields[0] == "uv" && fields[1] == "run" && fields[2] == "python":
		return "uv", fields[1:], true
	case len(fields) >= 3 && fields[0] == "python" && fields[1] == "-m" && fields[2] == "unittest":
		return "python", fields[1:], true
	default:
		return "", nil, false
	}
}

func splitCommandFields(command string) []string {
	rawFields := strings.Fields(command)
	fields := make([]string, 0, len(rawFields))
	for _, field := range rawFields {
		fields = append(fields, strings.Trim(field, `"'`))
	}
	return fields
}

func commandOutputLines(output []byte) []string {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return []string{}
	}

	lines := strings.Split(text, "\n")
	if len(lines) > 40 {
		lines = append(lines[:40], "... output truncated ...")
	}

	return lines
}

func buildAgentTaskResult(task AgentTask, failureMessage string) *AgentTaskResult {
	if task.Status == AgentTaskFailed || strings.TrimSpace(failureMessage) != "" {
		summary := strings.TrimSpace(failureMessage)
		if summary == "" {
			summary = "Agent task failed."
		}
		failureFile := detectFailureFile(task)
		suggestions := []string{
			"查看任务日志和最后一个失败 step 的 observation。",
			"保留当前任务分支，修正失败原因后重新创建任务。",
		}
		if task.Verification != nil && task.Verification.Command != "" {
			suggestions = append(suggestions, "在 workspace 中手动运行验证命令："+task.Verification.Command)
		}
		return &AgentTaskResult{
			Summary:         summary,
			FailureFile:     failureFile,
			NextSuggestions: suggestions,
		}
	}

	summaryParts := []string{}
	if task.BranchName != "" {
		summaryParts = append(summaryParts, "branch "+task.BranchName)
	}
	if len(task.ChangedFiles) > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("%d changed files", len(task.ChangedFiles)))
	} else {
		summaryParts = append(summaryParts, "no file changes")
	}
	if task.Verification != nil {
		summaryParts = append(summaryParts, "verification "+task.Verification.Status)
	}

	suggestions := []string{
		"检查生成文件是否符合项目风格。",
	}
	if task.Verification != nil && task.Verification.Status == "passed" {
		suggestions = append(suggestions, "验证通过后可以提交当前任务分支并发起合并。")
	}
	if len(task.ChangedFiles) > 0 {
		suggestions = append(suggestions, "重点 review changed files 中的 API client、类型和页面。")
	}

	return &AgentTaskResult{
		Summary:         strings.Join(summaryParts, ", "),
		NextSuggestions: suggestions,
	}
}

func detectFailureFile(task AgentTask) string {
	if task.Verification != nil {
		for _, line := range task.Verification.Output {
			if file := firstPathLikeToken(line); file != "" {
				return file
			}
		}
	}

	for index := len(task.Steps) - 1; index >= 0; index-- {
		step := task.Steps[index]
		if step.Observation.Status == "failed" || step.Observation.Status == "unsupported" {
			if step.Observation.Path != "" {
				return step.Observation.Path
			}
			if step.Action.Path != "" {
				return step.Action.Path
			}
		}
	}

	return ""
}

func firstPathLikeToken(line string) string {
	fields := strings.Fields(line)
	for _, field := range fields {
		cleaned := strings.Trim(field, "\"'`()[],:;")
		cleaned = stripLineColumnSuffix(cleaned)
		if strings.Contains(cleaned, "/") && hasCodeFileExtension(cleaned) {
			return cleaned
		}
	}
	return ""
}

func stripLineColumnSuffix(value string) string {
	parts := strings.Split(value, ":")
	for len(parts) > 1 && isDigits(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, ":")
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func hasCodeFileExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".py", ".js", ".jsx", ".ts", ".tsx", ".vue", ".css", ".json":
		return true
	default:
		return false
	}
}
