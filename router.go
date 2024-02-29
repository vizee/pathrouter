package pathrouter

import (
	"errors"
	"strings"
)

var (
	ErrConflict    = errors.New("Path conflict")
	ErrInvalidPath = errors.New("Invalid path")
)

type Param struct {
	Key   string
	Value string
}

type Params []Param

func (ps Params) Get(key string) (string, bool) {
	for i := range ps {
		if ps[i].Key == key {
			return ps[i].Value, true
		}
	}
	return "", false
}

type MatchResult[T any] struct {
	Params Params
	Value  T
}

type Router[T any] struct {
	root *node[T]
}

func (r *Router[T]) Match(path string, res *MatchResult[T]) bool {
	if r.root == nil {
		return false
	}

	cur := r.root
	for {
		switch cur.kind {
		case staticKind:
			if !strings.HasPrefix(path, cur.path) {
				return false
			}
			path = path[len(cur.path):]
		case paramKind:
			// 根节点是 param 或父节点标记 wildChild
			// param 要求 path 非空，捕获到下一个 / 之前
			if path == "" {
				return false
			}
			slash := strings.IndexByte(path, '/')
			if slash < 0 {
				slash = len(path)
			}
			res.Params = append(res.Params, Param{Key: cur.path[1:], Value: path[:slash]})
			path = path[slash:]
		case trailingKind:
			res.Params = append(res.Params, Param{Key: "*", Value: path})
			path = ""
		}

		// 如果 path 完成匹配，当前节点不是终止节点，考虑子节点存在 *
		if path == "" && (cur.end || !cur.wildChild) {
			break
		}

		if cur.wildChild {
			cur = cur.children[0]
		} else {
			found := false
			start := path[0]
			for i := 0; i < len(cur.indices); i++ {
				if cur.indices[i] == start {
					cur = cur.children[i]
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}

	if path == "" && cur.end {
		res.Value = cur.value
		return true
	}

	return false
}

func (r *Router[T]) Add(path string, value T) error {
	if r.root == nil {
		// 如果是一颗空树，直接设置根节点
		root := &node[T]{}
		err := root.init(path, value)
		if err != nil {
			return err
		}
		r.root = root
	}

	var orig *node[T]

	cur := r.root
	for {
		l := commonPrefixLength(cur.path, path)
		if l < len(cur.path) {
			// 分裂节点
			if cur.kind != staticKind {
				// 只有静态节点支持分裂
				return ErrConflict
			}

			child := &node[T]{
				kind:      cur.kind,
				end:       cur.end,
				wildChild: cur.wildChild,
				path:      cur.path[l:],
				indices:   cur.indices,
				children:  cur.children,
				value:     cur.value,
			}

			orig = cur
			cur = &node[T]{
				kind:      cur.kind,
				end:       false,
				wildChild: false,
				path:      orig.path[:l],
				indices:   child.path[:1],
				children:  []*node[T]{child},
			}
		}

		path = path[l:]

		// 路径完全匹配
		if path == "" {
			break
		}

		found := false
		start := path[0]
		for i := 0; i < len(cur.indices); i++ {
			if cur.indices[i] == start {
				cur = cur.children[i]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	err := cur.addSubPath(path, value)
	if err != nil {
		return err
	}

	if orig != nil {
		*orig = *cur
	}
	return nil
}

const (
	staticKind = iota
	paramKind
	trailingKind
)

type node[T any] struct {
	kind      uint8
	end       bool
	wildChild bool
	path      string
	indices   string
	children  []*node[T]
	value     T
}

func (n *node[T]) set(value T) {
	n.value = value
	n.end = true
}

func (n *node[T]) addChild(child *node[T]) error {
	switch n.kind {
	case trailingKind:
		return ErrInvalidPath
	case paramKind:
		// 只有 static 下可以跟随 param 和 *
		if child.kind != staticKind {
			return ErrInvalidPath
		} else if child.path[0] != '/' {
			// param 的 child 必须是 / 开头，否则说明是一个 param 被分裂
			return ErrConflict
		}
	}

	// 如果存在多个子节点，子节点必须同类，避免回退逻辑
	// 由于只有 static 可以分裂，param 和 * 的公共前缀会阻止节点有多个 param 或 * 子节点
	if len(n.children) > 0 && n.children[0].kind != child.kind {
		return ErrConflict
	}

	if child.kind != staticKind {
		n.wildChild = true
	}
	n.indices += child.path[:1]
	n.children = append(n.children, child)

	return nil
}

func (n *node[T]) init(path string, value T) error {
	seg, path := splitPathSegment(path)
	if seg == "*" {
		n.kind = trailingKind
	} else if strings.HasPrefix(seg, ":") {
		if len(seg) == 1 {
			return ErrInvalidPath
		}
		n.kind = paramKind
	} else {
		n.kind = staticKind
	}
	n.path = seg

	if path != "" {
		c := &node[T]{}
		err := c.init(path, value)
		if err != nil {
			return err
		}
		return n.addChild(c)
	}

	n.set(value)
	return nil
}

func (n *node[T]) addSubPath(path string, value T) error {
	if path == "" {
		n.set(value)
		return nil
	}

	child := &node[T]{}
	err := child.init(path, value)
	if err != nil {
		return err
	}

	return n.addChild(child)
}

func commonPrefixLength(a string, b string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

func splitPathSegment(path string) (string, string) {
	if path != "" {
		switch path[0] {
		case '*':
			return path[:1], path[1:]
		case ':':
			for i := 1; i < len(path); i++ {
				c := path[i]
				if c == '/' || c == ':' || c == '*' {
					return path[:i], path[i:]
				}
			}
		default:
			for i := 1; i < len(path); i++ {
				c := path[i]
				if c == ':' || c == '*' {
					return path[:i], path[i:]
				}
			}
		}
	}
	return path, ""
}
