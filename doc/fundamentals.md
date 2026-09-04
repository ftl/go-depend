# Definitions
Source: https://en.wikipedia.org/wiki/Software_package_metrics

## Instabillity (I)
The ratio of efferent coupling (`Ce`) to total coupling (`Ce + Ca`) such that `I = Ce / (Ce + Ca)`. This metric is an indicator of the package's resilience to change. The range for this metric is 0 to 1, with `I=0` indicating a completely stable package and `I=1` indicating a completely unstable package.

- **Efferent Coupling (outgoing dependencies)**: number of packages from the same module that this package depends on
- **Afferent Coupling (incoming dependencies)**: number of packages from the same module that depend on this package

## Abstractness (A)
The ratio of the number of abstract classes (and interfaces) in the analyzed package to the total number of classes in the analyzed package. The range for this metric is 0 to 1, with `A=0` indicating a completely concrete package and `A=1` indicating a completely abstract package.

- **Abstractions**: in Go: interfaces, generic types, generic functions
- **Implementations**: in Go: non-generic types, non-generic functions

go-depend counts only the exported declarations of a package. It counts the named types and the package-level functions. It does not count methods, constants, variables and type aliases. A method is a member of a type, and not a type. If a package has no exported types and no exported functions, its abstractness is 0.

## Distance from the Main Sequence (D)

The distance from the main sequence is `D = | A + I - 1 |`. The range for this metric is 0 to 1. A package with `D=0` is on the main sequence. A high value of `D` shows a bad balance between the abstractness and the instability of the package.

`D` uses the absolute value. Therefore `D` alone does not show which zone the package is in. Use the sign of `A + I - 1` to find the zone:

- a negative sign shows the zone of pain
- a positive sign shows the zone of uselessness

## Zone of Pain

A package is in the "zone of pain" if the value of `A + I - 1` is near `-1`. This is correct if both `A` and `I` are near `0`. Such a package contains very concrete implementations, and many other packages depend on it. A change to such a package causes changes in many downstream packages. A package in the zone of pain is difficult to maintain.

## Zone of Uselessness

A package is in the "zone of uselessness" if the value of `A + I - 1` is near `+1`. This is correct if both `A` and `I` are near `1`. Such a package defines abstractions, but almost no other package depends on them. At the same time, the package depends on many other packages. A change in one of these packages causes a change to the abstractions. A package in the zone of uselessness is difficult to maintain.
