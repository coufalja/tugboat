# tugboat
Tugboat is a hard fork of the [Dragonboat](https://github.com/lni/dragonboat) project.
Dragonboat is an awesome library and it is recommended to use instead of the Tugboat.
That being said there are a few notable differences.

## Project status
Project is still in very early stages of development going through some major API changes. It is not recommended for a production use.

## Differences from Dragonboat
* Simplified and removed some features
* Modularised (clean separation of logdb and transport component)
* Reduced the number of dependencies in the core package
* All the configuration is coming through the code (not from config files)
* Built for Go > 1.18 (Some APIs generified)
