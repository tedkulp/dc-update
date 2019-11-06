# dc-update

An opinioned script for updating large docker-compose based systems

## Why?

Docker and docker-compose are great. Though, my home server currently has 40 running containers, which is kind of a nightmare to keep on top of. This program is the next step in the evolution of a [gross Bash script](https://gist.github.com/tedkulp/80c5c707827556ed74e04bb92d924531) that I used to keep it all up to date.

## Install

    npm install -g dc-update

## Usage

To update all running containers in the docker-compose file in the current directory:

    dc-update

Only want to update a few containers?

    dc-update container1 container2 container3

Is the docker-compose file in another location?

    dc-update -f /path/to/docker-compose.yml

Of course, `--help` will give you the current list of options.

    dc-update --help

## Building Containers

In my particular setup, I have a few containers that have to be pulled and built before they can be updated.  Of course, we only want to restart a container if there is actually a reason to do so.  The `--build` option will pull and build the container first, and then try to update it along with the rest if it was updated and not just pulled from the cache.

    dc-update --build app --build proxy

__Note__: The build list and update list are completely separate.  If you want to build __AND__ update only a few containers, they must be specified twice.  (I may fix this later).

    dc-update --build app --build proxy app proxy
