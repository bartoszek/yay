package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"

	"github.com/Jguer/yay/v9/generic"
	alpm "github.com/jguer/go-alpm"
)

func questionCallback(question alpm.QuestionAny) {
	if qi, err := question.QuestionInstallIgnorepkg(); err == nil {
		qi.SetInstall(true)
	}

	qp, err := question.QuestionSelectProvider()
	if err != nil {
		return
	}

	if hideMenus {
		return
	}

	size := 0

	qp.Providers(alpmHandle).ForEach(func(pkg alpm.Package) error {
		size++
		return nil
	})

	fmt.Print(generic.Bold(generic.Cyan(":: ")))
	str := generic.Bold(fmt.Sprintf(generic.Bold("There are %d providers available for %s:"), size, qp.Dep()))

	size = 1
	var db string

	qp.Providers(alpmHandle).ForEach(func(pkg alpm.Package) error {
		thisDB := pkg.DB().Name()

		if db != thisDB {
			db = thisDB
			str += generic.Bold(generic.Cyan("\n:: ")) + generic.Bold("Repository "+db+"\n    ")
		}
		str += fmt.Sprintf("%d) %s ", size, pkg.Name())
		size++
		return nil
	})

	fmt.Println(str)

	for {
		fmt.Print("\nEnter a number (default=1): ")

		if config.NoConfirm {
			fmt.Println()
			break
		}

		reader := bufio.NewReader(os.Stdin)
		numberBuf, overflow, err := reader.ReadLine()

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}

		if overflow {
			fmt.Fprintln(os.Stderr, "Input too long")
			continue
		}

		if string(numberBuf) == "" {
			break
		}

		num, err := strconv.Atoi(string(numberBuf))
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s invalid number: %s\n", generic.Red("error:"), string(numberBuf))
			continue
		}

		if num < 1 || num > size {
			fmt.Fprintf(os.Stderr, "%s invalid value: %d is not between %d and %d\n", generic.Red("error:"), num, 1, size)
			continue
		}

		qp.SetUseIndex(num - 1)
		break
	}
}

func logCallback(level alpm.LogLevel, str string) {
	switch level {
	case alpm.LogWarning:
		fmt.Print(generic.Bold(generic.Yellow(generic.SmallArrow)), " ", str)
	case alpm.LogError:
		fmt.Print(generic.Bold(generic.Red(generic.SmallArrow)), " ", str)
	}
}
