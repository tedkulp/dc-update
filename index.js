const compose = require('docker-compose');
const path = require('path');
const Promise = require('bluebird');
const _ = require('lodash');
const ora = require('ora');
const Docker = require('dockerode');
const meow = require('meow');

const docker = new Docker();

const cli = meow(`
Usage
  $ dc-update [CONTAINER_NAME]...

Options
  --file, -f  Path to docker-compose.yml file

Examples
  $ dc-update influxdb grafana elasticsearch
    ...
`, {
  flags: {
    file: {
      type: 'string',
      alias: 'f',
    },
  },
});

const pathToDockerComposeFile = path.dirname(cli.flags.file) || process.cwd();
const dockerComposeOptions = {
  cwd: pathToDockerComposeFile,
  log: false,
};

const getServiceNames = async () => {
  return compose.configServices(dockerComposeOptions).then(result => {
    return result.out.split(/\n/).map(service => {
      return _.trim(service);
    }).filter(service => service);
  });
};

const getCurrentContainerId = async containerName => {
  return compose.ps({
    ...dockerComposeOptions,
    commandOptions: ['-q', containerName],
  }).then(result => {
    return _.trim(result.out);
  });
};

const getLatestImageId = async (containerName, containerObj) => {
  await compose.pullOne(containerName, {
    ...dockerComposeOptions,
    commandOptions: ['-q'],
  }).then(result => {
    return _.trim(result.out);
  });

  return containerObj.inspect().then(res => {
    return _.get(res, 'Config.Image', '');
  }).then(imageName => {
    if (imageName) {
      return docker.listImages({
        filters: {
          reference: [imageName],
        }
      }).then(ret => {
        return ret.reduce((acc, imageInfo) => {
          if (imageInfo.Created > acc.Created) {
            return imageInfo;
          } else {
            return acc;
          }
        }, { Created: 0 });
      });
    }
  }).then(latestImageInfo => {
    return _.get(latestImageInfo, 'Id', 'sha256:').split(':')[1];
  });
};

const getCurrentImageId = async containerObj => {
  return containerObj.inspect().then(res => {
    return _.get(res, 'Image', 'sha256:').split(':')[1];
  });
};

const restartContainer = async containerName => {
  await compose.stopOne(containerName, dockerComposeOptions);
  await compose.rmOne(containerName, dockerComposeOptions);
  return compose.upOne(containerName, dockerComposeOptions);
};

const updateContainer = async containerName => {
  const spinner = ora(`Updating ${containerName}`).start();

  const currentContainerId = await getCurrentContainerId(containerName);
  if (!currentContainerId) {
    spinner.warn(`${containerName} is not running`);
    return;
  }

  const currentContainer = docker.getContainer(currentContainerId);
  const currentImageId = await getCurrentImageId(currentContainer);
  const latestImageId = await getLatestImageId(containerName, currentContainer);

  if (latestImageId && currentImageId !== latestImageId) {
    // if (latestImageId && currentImageId === latestImageId) {
    spinner.text = `Updating and restarting ${containerName}`;
    await restartContainer(containerName).catch(err => {
      spinner.fail(`Failed to restart ${containerName}`);
      console.error(err);
    });
    spinner.succeed(`Updated ${containerName}`);
  } else {
    spinner.succeed(`${containerName} is already up to date`);
  }
};

(async () => {
  let serviceNames = cli.input;
  if (serviceNames.length === 0) {
    serviceNames = await getServiceNames();
  }

  Promise.mapSeries(serviceNames, async service => {
    await updateContainer(service);
  });
})();
