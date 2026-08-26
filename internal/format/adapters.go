package format

import (
	"strings"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// mavenAdapter implements groupId:artifactId[:version].
type mavenAdapter struct{}

func (mavenAdapter) Name() string { return "maven" }

func (mavenAdapter) parse(coord string, requireName bool) (Ref, error) {
	parts := strings.Split(coord, ":")
	if len(parts) > 3 || len(parts) == 0 {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"invalid maven coordinate %q (expected groupId:artifactId[:version])", coord)
	}
	for i, p := range parts {
		if p == "" {
			return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
				"invalid maven coordinate %q: empty segment at position %d", coord, i+1)
		}
	}
	ref := Ref{Group: parts[0], Name: parts[1]}
	if len(parts) == 3 {
		ref.Version = parts[2]
	} else if len(parts) < 2 && requireName {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"invalid maven coordinate %q: missing artifactId", coord)
	}
	return ref, nil
}

func (a mavenAdapter) ParseSearch(query string) (SearchParams, error) {
	if !strings.Contains(query, ":") {
		return SearchParams{Q: query}, nil
	}
	ref, err := a.parse(query, true)
	if err != nil {
		return SearchParams{}, err
	}
	return SearchParams{Group: ref.Group, Name: ref.Name, Version: ref.Version}, nil
}

func (mavenAdapter) ParseComponent(coord string, requireVersion bool) (Ref, error) {
	if !strings.Contains(coord, ":") {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"invalid maven coordinate %q: missing groupId separator (expected groupId:artifactId[:version])", coord)
	}
	ref, err := mavenAdapter{}.parse(coord, true)
	if err != nil {
		return Ref{}, err
	}
	if requireVersion && ref.Version == "" {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"missing version in %q (expected groupId:artifactId:version)", coord)
	}
	return ref, nil
}

// simpleAdapter implements name[@version] for npm/pypi/cargo/go.
type simpleAdapter struct{ format string }

func (s simpleAdapter) Name() string { return s.format }

func splitNameVersion(coord string) (string, string, bool) {
	idx := strings.LastIndex(coord, "@")
	if idx <= 0 || idx == len(coord)-1 {
		return "", "", false
	}
	return coord[:idx], coord[idx+1:], true
}

func (simpleAdapter) ParseSearch(query string) (SearchParams, error) {
	name, version, ok := splitNameVersion(query)
	if !ok {
		return SearchParams{Q: query}, nil
	}
	return SearchParams{Name: name, Version: version}, nil
}

func (s simpleAdapter) ParseComponent(coord string, requireVersion bool) (Ref, error) {
	name, version, ok := splitNameVersion(coord)
	if !ok {
		name = coord
	}
	if strings.TrimSpace(name) == "" {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"invalid %s component %q (expected name[@version])", s.format, coord)
	}
	if requireVersion && version == "" {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"missing version in %q; use `nx %s versions %s` to list available versions", coord, s.format, coord)
	}
	return Ref{Name: name, Version: version}, nil
}

// dockerAdapter implements image[:tag]. A colon only separates the tag when
// it appears after the last slash so registry hosts with ports stay intact.
type dockerAdapter struct{}

func (dockerAdapter) Name() string { return "docker" }

// splitImageTag separates image from tag. A colon only starts a tag when it
// appears after the last slash so registry hosts with ports stay intact.
func splitImageTag(image string) (string, string) {
	colon, slash := strings.LastIndex(image, ":"), strings.LastIndex(image, "/")
	if colon > slash+1 && colon != len(image)-1 {
		return image[:colon], image[colon+1:]
	}
	return image, ""
}

func (dockerAdapter) ParseSearch(query string) (SearchParams, error) {
	image, tag := splitImageTag(query)
	if image == "" {
		return SearchParams{}, nxerrors.New(nxerrors.ClassCoordinate, "empty docker image %q", query)
	}
	if tag != "" {
		return SearchParams{Name: image, Version: tag}, nil
	}
	return SearchParams{Q: image}, nil
}

func (dockerAdapter) ParseComponent(coord string, requireVersion bool) (Ref, error) {
	image, tag := splitImageTag(coord)
	if image == "" {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate, "empty docker image %q", coord)
	}
	if requireVersion && tag == "" {
		return Ref{}, nxerrors.New(nxerrors.ClassCoordinate,
			"missing tag in %q; use `nx docker versions %s` to list available tags", coord, coord)
	}
	return Ref{Name: image, Version: tag}, nil
}
