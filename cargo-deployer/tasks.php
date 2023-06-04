#!/usr/bin/env php
<?php

echo '['.date('Y-m-d H:i:s').'] Start'.PHP_EOL;
sleep(1);

declare(ticks=1);
pcntl_signal(SIGTERM, function () {
	exit(0);
});

new Tasks;

class Tasks {
	const EACH_ITER = 8;
	const DEF_ENDPOINT = 'https://host.docker.internal:8443';
	public int $iteration = 0;
	public array $sinceLogs = [];

	public function __construct() {
		while (true) {
			++$this->iteration;
			$result = $this->push();

			if (($result['ok'] ?? false) === true) {
				if (!empty($result['data']['execs'])) {
					echo '['.date('Y-m-d H:i:s').'] Received data for deployments'.PHP_EOL;
					$this->containersExecute($result['data']['execs']);
					echo '['.date('Y-m-d H:i:s').'] End deployments'.PHP_EOL;

					$this->push();
				}

			} else {
				echo '['.date('Y-m-d H:i:s').'] '.print_r($result, true).PHP_EOL;
			}

			sleep(self::EACH_ITER);
		}

	}

	public function endpoint(): string {
		return $_SERVER['ENDPOINT'] ?? self::DEF_ENDPOINT;
	}

	public function namespaceShip(): string {
		return $_SERVER['NAMESPACE'] ?? 'ctr-ship';
	}

	public function namespaceDeployment($name = ''): string {
		return $this->namespaceShip().'.deployment'.(!empty($name) ? '='.$name : '');
	}

	public function push() {
		$curl = curl_init($this->endpoint().'/nodes');
		curl_setopt($curl, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
		curl_setopt($curl, CURLOPT_CONNECTTIMEOUT, 5);
		curl_setopt($curl, CURLOPT_TIMEOUT, 10);
		curl_setopt($curl, CURLOPT_POST, true);
		curl_setopt($curl, CURLOPT_POSTFIELDS, json_encode([
			'iteration'  => $this->iteration,
			'uptime'     => trim(shell_exec('uptime')),
			'containers' => $this->containers()
		]));
		if (self::endpoint() === self::DEF_ENDPOINT) {
			curl_setopt($curl, CURLOPT_SSL_VERIFYPEER, false);
			curl_setopt($curl, CURLOPT_SSL_VERIFYHOST, false);
		}
		curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
		$json = curl_exec($curl);
		$error = curl_error($curl);
		$code = curl_getinfo($curl, CURLINFO_HTTP_CODE);
		$message = curl_error($curl);
		curl_close($curl);

		if ($code >= 500) {
			return [
				'status'  => 'failed',
				'code'    => $code,
				'message' => $message,
				'result'  => json_decode($json, true)
			];
		}

		if (empty($json)) {
			error_log($error);
			return [
				'status'  => 'failed',
				'code'    => $code,
				'message' => 'empty-response'
			];
		}

		return json_decode($json, true);
	}

	public function requestDocker(string $endpoint): string {
		$curl = curl_init($endpoint);
		curl_setopt($curl, CURLOPT_CONNECTTIMEOUT, 5);
		curl_setopt($curl, CURLOPT_TIMEOUT, 10);
		curl_setopt($curl, CURLOPT_UNIX_SOCKET_PATH, '/var/run/docker.sock');
		curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
		$json = curl_exec($curl);
		$code = curl_getinfo($curl, CURLINFO_HTTP_CODE);
		curl_close($curl);

		if ($code >= 500) {
			error_log(json_encode([
				'curlDocker' => $endpoint,
				'code'       => $code,
				'result'     => $json
			]));
		}

		return $json;
	}

	public function containers(): array {
		$json = $this->requestDocker(
			'http://localhost/v1.40/containers/json?filters='.json_encode([
				'label' => [
					$this->namespaceDeployment()
				]
			]));

		if (empty($json)) {
			return [];
		}

		$containers = [];
		foreach (json_decode($json, true) as $e) {
			$logs = [];
			if (!isset($this->sinceLogs[$e['Id']])) {
				$this->sinceLogs[$e['Id']] = time();
			}
			$logsText = $this->requestDocker(
				'http://localhost/v1.40/containers/'.$e['Id'].'/logs'
				.'?since='.$this->sinceLogs[$e['Id']]
				.'&stdout=true&stderr=true&timestamps=true');

			foreach (explode(PHP_EOL, $logsText) as $str) {
				if (!isset($str[0]))
					continue;

				$offset = 0;
				$std = (int)bin2hex($str[0]);
				// stdout || stderr
				if ($std === 0x01 || $std === 0x02) {
					$offset = 8;
				}

				if (!$time = substr($str, $offset, 30))
					continue;

				$logs[] = [
					'time' => $time,
					'mess' => substr($str, $offset + 31),
				];
			}
			$this->sinceLogs[$e['Id']] = time();

			$containers[] = [
				'id'           => $e['Id'],
				'idShort'      => substr($e['Id'], 0, 12),
				'name'         => substr($e['Names'][0], 1),
				'imageId'      => $e['ImageID'],
				'imageIdShort' => substr($e['ImageID'], 7, 12),
				'labels'       => $e['Labels'],
				'state'        => $e['State'],
				'status'       => $e['Status'],
				'logs'         => $logs,
			];
		}

		return $containers;
	}

	public function containerCommand($e, $deploymentName): string {
		$params = '';

		if (!empty($e['runtime']) && $e['runtime'] == 'nvidia'
			&& !empty(trim(shell_exec('lspci | grep -i vga | grep -i nvidia')))) {
			$params .= ' --runtime=nvidia';
		}
		if (!empty($e['pid'])) {
			$params .= ' --pid='.$e['pid'];
		}
		if (!empty($e['privileged'])) {
			$params .= ' --privileged';
		}
		if (!empty($e['network'])) {
			$params .= ' --network '.$e['network'];
		}
		if (!empty($e['restart'])) {
			$params .= ' --restart '.$e['restart'];
		}
		if (!empty($e['caps'])) {
			$params .= ' --cap-add='.implode(' --cap-add=', $e['caps']);
		}
		if (!empty($e['hosts'])) {
			$params .= ' --add-host='.implode(' --add-host=', $e['hosts']);
		}
		if (!empty($e['ports'])) {
			$params .= ' -p '.implode(' -p ', $e['ports']);
		}
		if (!empty($e['mounts'])) {
			$params .= ' --mount '.implode(' --mount ', $e['mounts']);
		}
		if (!empty($e['volumes'])) {
			$params .= ' -v '.implode(' -v ', $e['volumes']);
		}
		if (!empty($e['environment'])) {
			$params .= ' -e '.implode(' -e ', $e['environment']);
		}
		if (!empty($e['entrypoint'])) {
		    $params .= ' --entrypoint '.$e['entrypoint'];
		}

		return 'docker run -d'
			.' --name '.$e['name']
			.' --label '.$this->namespaceDeployment($deploymentName)
			.' --log-driver json-file'
			.' --log-opt max-size=5m'
			.' '.$params
			.' '.$e['from']
			.(!empty($e['command']) ? ' '.$e['command'] : '');
	}

	public function selfUpgrade($d) {
		echo '• Run self upgrade'.PHP_EOL;

		if (empty($d['containers'][0])) {
			echo 'Error: container manifest is undefined'.PHP_EOL;

			return;
		}

		$dc = $d['containers'][0];

		echo '• Pulling "'.$dc['from'].'": ';
		echo shell_exec('docker pull '.$dc['from']).PHP_EOL;

		$oldId = shell_exec('docker ps -aqf=name='.$dc['name']);
		if (!empty($oldId)) {
			echo '• Container rename'.PHP_EOL;
			shell_exec('docker rename '.$dc['name'].' '.$dc['name'].'-old');
		}

		echo '• Container run: ';
		echo shell_exec($this->containerCommand($dc, $d['deploymentName']).' 2>&1').PHP_EOL;

		if (!empty($oldId)) {
			echo '• Container '.$dc['name'].'-old rm force: ';
			echo shell_exec('docker rm -f '.$dc['name'].'-old 2>&1').PHP_EOL;
		}

		exit;
	}

	public function destroy() {
		$ids = explode(PHP_EOL,
			trim(shell_exec('docker ps -a --filter "label='
				.$this->namespaceDeployment().'" --format "{{.Names}}"'))
		);
		if (!empty($ids)) {
			$spaceCargo = $this->namespaceShip().'.cargo-deployer';
			$ids = array_flip($ids);
			unset($ids[$spaceCargo]);
			$ids = implode(' ', array_flip($ids));

			echo '• Containers destroy: ';
			echo shell_exec('docker rm -f '.$ids.' 2>&1').PHP_EOL;

			echo '• Images prune: ';
			echo shell_exec('docker image prune -af').PHP_EOL;

			echo '• Self destroy: ';
			echo shell_exec('docker rm -f '.$spaceCargo.' 2>&1').PHP_EOL;
		}
		exit;
	}

	public function containersExecute($data) {
		$data = json_decode($data, true);

		if (empty($data)) {
			echo '• Empty data options'.PHP_EOL;

			return;
		}

		foreach ($data as $d) {
			if ($d['selfUpgrade']) {
				$this->selfUpgrade($d);
			}

			if ($d['destroy'] === true && empty($d['deploymentName'])) {
				$this->destroy();
			}

			echo '••••••••••'.PHP_EOL;
			echo '• Execution deployment: '.$d['deploymentName'].PHP_EOL;

			if (!isset($d['containers']))
				$d['containers'] = [];

			if ($d['destroy'] === false) {
				foreach ($d['containers'] as $e) {
					echo '• Pulling "'.$e['from'].'": ';
					echo shell_exec('docker pull '.$e['from']).PHP_EOL;
				}
			}

			$ids = str_replace(PHP_EOL, ' ',
				shell_exec('docker ps -aqf "label='.$this->namespaceDeployment($d['deploymentName']).'"')
			);
			if (!empty($ids)) {
				$time = !empty($e['stop-time']) ? '--time '.(int)$e['stop-time'].' ' : '';
				echo '• Containers stop'.($time != '' ? ' '.$time.'sec.' : '').': ';
				echo shell_exec('docker stop '.$time.$ids.' 2>&1').PHP_EOL;
				echo '• Containers rm: ';
				echo shell_exec('docker rm '.$ids.' 2>&1').PHP_EOL;
			}

			if ($d['destroy'] === false) {
				foreach ($d['containers'] as $e) {
					echo '• Container run "'.$e['name'].'": ';
					echo shell_exec($this->containerCommand($e, $d['deploymentName']).' 2>&1').PHP_EOL;

					if (!empty($e['webhook'])) {
						echo '• Webhook touch "'.$e['webhook'].'": ';
						echo file_get_contents($e['webhook']).PHP_EOL;
					}

					if (!empty($e['executions'])) {
						echo '• Executions in container'.PHP_EOL;
						foreach ($e['executions'] as $x) {
							echo shell_exec('docker exec '.$e['name'].' '.$x);
						}
					}
				}
			}

			echo '••••••••••'.PHP_EOL;
		}

		echo '• Image prune'.PHP_EOL;
		echo shell_exec('docker image prune -f').PHP_EOL;
	}
}
