package cli

import (
	"fmt"
	"path"

	"github.com/pkg/errors"

	"github.com/PlanktoScope/forklift/internal/app/forklift"
)

func PrintPalletImports(indent int, pallet *forklift.FSPallet) error {
	imps, err := pallet.LoadImports("**/*")
	if err != nil {
		return err
	}
	for _, imp := range imps {
		IndentedPrintf(indent, "%s\n", imp.Name)
	}
	return nil
}

func PrintImportInfo(
	indent int, pallet *forklift.FSPallet, cache forklift.PathedPalletCache, importName string,
) error {
	imp, err := pallet.LoadImport(importName)
	if err != nil {
		return errors.Wrapf(
			err, "couldn't find import group declaration %s in pallet %s", importName, pallet.FS.Path(),
		)
	}
	resolved, err := forklift.ResolveImport(pallet, cache, imp)
	if err != nil {
		return errors.Wrapf(err, "couldn't resolve import group %s", imp.Name)
	}
	if err = PrintResolvedImport(indent, resolved); err != nil {
		return errors.Wrapf(err, "couldn't print resolved import group %s", imp.Name)
	}
	return nil
}

func PrintResolvedImport(
	indent int, resolved *forklift.ResolvedImport,
) error {
	if err := printImport(indent, resolved); err != nil {
		return err
	}
	return nil
}

func printImport(indent int, imp *forklift.ResolvedImport) error {
	IndentedPrint(indent, "Import group")
	if imp.Import.Def.Disabled {
		fmt.Print(" (disabled!)")
	}
	fmt.Printf(": %s\n", imp.Name)
	indent++

	IndentedPrintf(indent, "Import source: %s\n", imp.Pallet.Path())

	printModifiers(indent, imp.Def.Modifiers)

	return nil
}

func printModifiers(indent int, modifiers []forklift.ImportModifier) {
	IndentedPrint(indent, "Group modifiers:")
	if len(modifiers) == 0 {
		fmt.Print(" (none)")
	}
	fmt.Println()
	indent++
	for i, modifier := range modifiers {
		switch modifier.Type {
		case "add":
			printAddModifier(indent, i, modifier)
		case "remove":
			printRemoveModifier(indent, i, modifier)
		default:
			BulletedPrintf(indent, "[%d] Unknown modifier type %s: %+v\n", i, modifier.Type, modifier)
		}
	}
}

func printAddModifier(indent, index int, modifier forklift.ImportModifier) {
	BulletedPrintf(indent, "[%d] Add files to group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		if modifier.Source == modifier.Target {
			BulletedPrintf(indent, "Add: %s\n", path.Join(modifier.Target, filter))
			continue
		}
		BulletedPrintf(indent, "From source: %s\n", path.Join(modifier.Source, filter))
		IndentedPrintf(indent+1, "Add as:      %s\n", path.Join(modifier.Target, filter))
	}
}

func printRemoveModifier(indent, index int, modifier forklift.ImportModifier) {
	BulletedPrintf(indent, "[%d] Remove files from group", index)
	if modifier.Description == "" {
		fmt.Println()
	} else {
		fmt.Printf(": %s\n", modifier.Description)
	}
	indent++
	indent++ // Because we're nesting a bulleted list in another bulleted list
	for _, filter := range modifier.OnlyMatchingAny {
		BulletedPrintf(indent, "Remove: %s\n", path.Join(modifier.Target, filter))
	}
}
