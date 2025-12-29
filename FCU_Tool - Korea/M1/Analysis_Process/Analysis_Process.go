package Analysis_Process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"FCU_Tools/M1/M1_Public_Data"
	"FCU_Tools/M1/System_Analysis"
)

// 1단계는 고정되어 있으며, BuildDir/<Model>/simulink/systems/system_root.xml만 분석합니다.
func RunAnalysis(maxDepth int) {

	buildRoot := M1_Public_Data.BuildDir
	if buildRoot == "" {
		fmt.Println("❌ BuildDir이 비어 있습니다. 먼저 SetWorkDir()를 호출하여 작업 공간을 초기화하세요.")
		return
	}

	// BuildDir 하위의 모델 디렉터리
	modelDirs, err := os.ReadDir(buildRoot)
	if err != nil {
		fmt.Println("❌ BuildDir 디렉터리를 읽을 수 없습니다：", err)
		return
	}

	for _, modelEntry := range modelDirs {
		if !modelEntry.IsDir() {
			continue
		}

		modelName := modelEntry.Name()
		modelPath := filepath.Join(buildRoot, modelName)

		// 고정된 구조: <BuildDir>/<Model>/simulink/systems/system_root.xml
		sysDir := filepath.Join(modelPath, "simulink", "systems")
		xmlPath := filepath.Join(sysDir, "system_root.xml")

		if _, err := os.Stat(xmlPath); err != nil {
			continue // 모델에 system_root.xml이 없으면 건너뜁니다.
		}

		fmt.Printf("🔍 모델 분석 [%s] (최대 깊이: %d)\n", modelName, maxDepth)

		// 재귀 분석을 시작하며, 1층(L1)부터 수행합니다. L1에는 부모 노드가 없습니다.
		err = analyzeRecursive(sysDir, "system_root.xml", 1, maxDepth, "")
		if err != nil {
			fmt.Println("❌ 분석 실패：", err)
			continue
		}
	}

	fmt.Printf("✅ 분석 완료 (최대 깊이: %d)\n", maxDepth)
}

// 재귀 분석 함수로, maxDepth에 따라 재귀 깊이를 제어합니다.
// fatherName: 현재 레벨의 System에 해당하는 ‘부모 노드 이름’이며, 다음 레벨에서 FatherNode 정보를 출력할 때 사용합니다.
// dir는 분석 경로, file은 분석할 파일, currentLevel은 현재 분석 레벨, maxDepth는 분석할 최대 레벨(깊이)입니다.
// fatherName은 상위(부모) 분석 대상의 이름을 의미하며, 예를 들어 system4.ldi.xml과 같이 상위 파일명을 전달합니다.
func analyzeRecursive(dir, file string, currentLevel, maxDepth int, fatherName string) error {
	// 현재 레벨이 최대 깊이를 초과하면 재귀를 중단합니다.
	if currentLevel > maxDepth {
		return nil
	}

	// 통합 진입점으로, System_Analysis가 level에 따라 필터링 로직을 결정합니다.
	subsystems, err := System_Analysis.AnalyzeSubSystemsInFile(dir, file, currentLevel, fatherName)
	if err != nil {
		return err
	}

	// 다음 레벨을 재귀적으로 분석합니다.
	if len(subsystems) > 0 && currentLevel < maxDepth {
		nextLevel := currentLevel + 1
		for _, sub := range subsystems {
			nextFile := fmt.Sprintf("system_%s.xml", sub.SID)
			nextFull := filepath.Join(dir, nextFile)

			if _, err := os.Stat(nextFull); err == nil {
				// 다음 레벨의 부모 노드 = 현재 레벨의 서브시스템 이름
				nextFather := strings.TrimSpace(sub.Name)
				if err := analyzeRecursive(dir, nextFile, nextLevel, maxDepth, nextFather); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
