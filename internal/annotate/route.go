package annotate

import (
	"fmt"
	"regexp"
	"strings"
)

func ParseRouteComment(comment string) (RouteSpec, error) {
	// comment: "@Router /path [get]" ou "@Route ..."
	// comportamento atual: usa o texto depois de "@Router"
	// (mantemos compatível com o core original)
	lower := strings.ToLower(strings.Fields(strings.TrimSpace(comment))[0])
	if lower != "@router" && lower != "@route" {
		return RouteSpec{}, fmt.Errorf("not a route comment: %s", comment)
	}

	// mantém a regra antiga: corta a partir do literal "@Router"
	// Para @Route, preferimos também aceitar cortando pela palavra inicial.
	sourceString := strings.TrimSpace(comment[len(strings.Fields(comment)[0]):])

	// /path [method]
	re := regexp.MustCompile(`([\w\.\/\-{}]+)[^\[]+\[([^\]]+)`)
	matches := re.FindStringSubmatch(sourceString)
	if len(matches) != 3 {
		return RouteSpec{}, fmt.Errorf("can not parse router comment %q", comment)
	}

	return RouteSpec{
		Path:   matches[1],
		Method: matches[2],
	}, nil
}
