package main

import (
    "bufio"
    "fmt"
    "os"
    "regexp"
    "strings"
)

type LineInfo struct {
    Original string
    Parsed   string
    IsRule   bool
}

func main() {
    inputFile, err := os.Open("rule-set/realip.list")
    if err != nil {
        fmt.Println("Error opening input file:", err)
        return
    }
    defer inputFile.Close()

    outputFile, err := os.Create("output.list")
    if err != nil {
        fmt.Println("Error creating output file:", err)
        return
    }
    defer outputFile.Close()

    writer := bufio.NewWriter(outputFile)
    defer writer.Flush()

    var lines []LineInfo

    scanner := bufio.NewScanner(inputFile)
    for scanner.Scan() {
        rawText := scanner.Text()
        line := strings.TrimSpace(rawText)

        if strings.HasPrefix(line, "#") {
            continue
        }

        if line == "" {
            lines = append(lines, LineInfo{Original: rawText, IsRule: false})
            continue
        }

        var result string
        if line == "*" {
            result = "DOMAIN-REGEX,^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$"
        } else if !strings.ContainsAny(line, "*+") && !strings.HasPrefix(line, ".") {
            result = fmt.Sprintf("DOMAIN,%s", line)
        } else if strings.HasPrefix(line, "+.") && !strings.ContainsAny(line[2:], "*+") {
            result = fmt.Sprintf("DOMAIN-SUFFIX,%s", line[2:])
        } else {
            prefix := "^"
            pattern := line

            if strings.HasPrefix(line, "+.") {
                prefix = `(^|\.)`
                pattern = line[2:]
            } else if strings.HasPrefix(line, ".") {
                prefix = `\.`
                pattern = line[1:]
            }

            pattern = regexp.QuoteMeta(pattern)
            pattern = strings.ReplaceAll(pattern, `\*`, "[^.]+")
            pattern = strings.ReplaceAll(pattern, `\+`, ".+")

            result = fmt.Sprintf("DOMAIN-REGEX,%s%s$", prefix, pattern)
        }

        lines = append(lines, LineInfo{
            Original: rawText,
            Parsed:   result,
            IsRule:   true,
        })
    }

    if err := scanner.Err(); err != nil {
        fmt.Println("Error reading input file:", err)
        return
    }

    finalPositions := optimizeAndMapPositions(lines)

    for i, line := range lines {
        if !line.IsRule {
            fmt.Fprintln(writer, line.Original)
        } else {
            if finalRule, ok := finalPositions[i]; ok {
                fmt.Fprintln(writer, finalRule)
            }
        }
    }

    fmt.Println("Save to output.list")
}

func optimizeAndMapPositions(lines []LineInfo) map[int]string {
    uniqueMap := make(map[string]int)
    for i, line := range lines {
        if !line.IsRule {
            continue
        }
        if _, exists := uniqueMap[line.Parsed]; !exists {
            uniqueMap[line.Parsed] = i
        }
    }

    type groupInfo struct {
        depths    map[int]struct{}
        originals []string
        minIdx    int
    }
    groups := make(map[string]*groupInfo)
    finalPositions := make(map[int]string)

    for rule, idx := range uniqueMap {
        if strings.HasPrefix(rule, "DOMAIN-REGEX,") && strings.HasSuffix(rule, "$") {
            core := rule[:len(rule)-1]
            depth := 0
            suffix := `\.[^.]+`

            for strings.HasSuffix(core, suffix) {
                core = strings.TrimSuffix(core, suffix)
                depth++
            }

            if depth > 0 {
                if groups[core] == nil {
                    groups[core] = &groupInfo{
                        depths:    make(map[int]struct{}),
                        originals: []string{},
                        minIdx:    idx,
                    }
                }
                groups[core].depths[depth] = struct{}{}
                groups[core].originals = append(groups[core].originals, rule)

                if idx < groups[core].minIdx {
                    groups[core].minIdx = idx
                }
                continue
            }
        }

        finalPositions[idx] = rule
    }

    for core, info := range groups {
        if len(info.depths) > 1 {
            simplifiedRule := core + `\..`
            finalPositions[info.minIdx] = simplifiedRule
        } else {
            for _, orig := range info.originals {
                origIdx := uniqueMap[orig]
                finalPositions[origIdx] = orig
            }
        }
    }

    return finalPositions
}