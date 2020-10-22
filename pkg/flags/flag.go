package flags

import (
	"flag"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"
)

func SetFlagsFromEnv(lg *zap.Logger, prefix string, fs *flag.FlagSet) error {
	var err error
	alreadySet := make(map[string]bool)
	//这个是将已经设置的flag记录下
	fs.Visit(func(f *flag.Flag) {
		alreadySet[f.Name] = true
	})

	usedEnvSet := make(map[string]bool)
	//这里将要访问所有的flag集
	fs.VisitAll(func(f *flag.Flag) {
		if serr := setFlagFromEnv(lg, fs, prefix, f.Name, alreadySet, usedEnvSet, true); serr != nil {
			err = serr
		}
	})

	verifyEnv(lg, prefix, usedEnvSet, alreadySet)

	return nil
}

type flagSetter interface {
	Set(fk string, fv string) error
}

//开始验证环境变量，有必要么？  取出所有环境变量然后和etcd的对比？
func verifyEnv(lg *zap.Logger, prefix string, usedEnvKey, alreadySet map[string]bool) {
	for _, env := range os.Environ() {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			if lg != nil {
				lg.Warn("found invalid environment variable", zap.String("environment-variable", env))
			}
		}
		if usedEnvKey[kv[0]] {
			continue
		}
		if alreadySet[kv[0]] {
			if lg != nil {
				lg.Fatal(
					"conflicting environment variable is shadowed by corresponding command-line flag (either unset environment variable or disable flag))",
					zap.String("environment-variable", kv[0]),
				)
			}
		}

		if strings.Contains(env, prefix+"_") {
			if lg != nil {
				lg.Warn("unrecognized environment variable", zap.String("environment-variable", env))
			}
		}
	}
}

//如果flag已经被设置 则略过，否则取环境变量赋值给flag
func setFlagFromEnv(lg *zap.Logger, fs flagSetter, prefix, fname string, alreadySet, usedEnvKey map[string]bool, log bool) error {
	key := FlagToEnv(prefix, fname)
	if !alreadySet[key] {
		keyVal := os.Getenv(key)
		if keyVal != "" {
			usedEnvKey[key] = true
			serr := fs.Set(fname, keyVal)
			if serr != nil {
				return fmt.Errorf("invalid value %q for %s: %v", keyVal, key, serr)
			}
			if log && lg != nil {
				lg.Info(
					"recognized and used environment variable",
					zap.String("variable-name", key),
					zap.String("variable-value", keyVal),
				)
			}
		}
	}

	return nil
}

func FlagToEnv(prefix, name string) string {
	return strings.ToUpper(prefix) + "_" + strings.ToUpper(strings.Replace(name, "-", "_", -1))
}

//原实现中没直接使用lookup因为lookup的寻找包括了未设置的flag名字
//visit则是只访问到已经设置的flag
func IsSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
