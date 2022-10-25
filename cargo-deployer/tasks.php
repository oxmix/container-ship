#!/usr/bin/env php
<?php

echo '['.date('Y-m-d H:i:s').'] Start'.PHP_EOL;
sleep(1);

$timer = 8;
$iteration = 0;

declare(ticks=1);
pcntl_signal(SIGTERM, function () {
	exit(0);
});

function endpoint(): string {
	return $_SERVER['ENDPOINT'] ?? 'https://host.docker.internal:8443';
}

function namespaceShip(): string {
	return $_SERVER['NAMESPACE'] ?? 'ctr-ship';
}

function namespaceDeployment($name = ''): string {
	return namespaceShip().'.deployment'.(!empty($name) ? '='.$name : '');
}

function push($iteration) {
	$curl = curl_init(endpoint().'/nodes');
	curl_setopt($curl, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
	curl_setopt($curl, CURLOPT_CONNECTTIMEOUT, 5);
	curl_setopt($curl, CURLOPT_TIMEOUT, 10);
	curl_setopt($curl, CURLOPT_POST, true);
	curl_setopt($curl, CURLOPT_POSTFIELDS, json_encode([
		'iteration'  => $iteration,
		'uptime'     => trim(shell_exec('uptime')),
		'containers' => containers()
	]));
	curl_setopt($curl, CURLOPT_SSL_VERIFYPEER, false);
	curl_setopt($curl, CURLOPT_SSL_VERIFYHOST, false);
	curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
	$json = curl_exec($curl);
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
		return [
			'status'  => 'failed',
			'code'    => $code,
			'message' => 'empty-response'
		];
	}

	return json_decode($json, true);
}

function containers(): array {
	$curl = curl_init('http://localhost/v1.40/containers/json?filters='.json_encode([
			'label' => [
				namespaceDeployment()
			]
		]));
	curl_setopt($curl, CURLOPT_CONNECTTIMEOUT, 5);
	curl_setopt($curl, CURLOPT_TIMEOUT, 10);
	curl_setopt($curl, CURLOPT_UNIX_SOCKET_PATH, '/var/run/docker.sock');
	curl_setopt($curl, CURLOPT_RETURNTRANSFER, true);
	$json = curl_exec($curl);
	$code = curl_getinfo($curl, CURLINFO_HTTP_CODE);
	curl_close($curl);

	if ($code > 200 || empty($json)) {
		return [
			'ok'     => false,
			'code'   => $code,
			'result' => $json
		];
	}

	$containers = [];
	foreach (json_decode($json, true) as $e) {
		$containers[] = [
			'id'           => $e['Id'],
			'idShort'      => substr($e['Id'], 0, 12),
			'name'         => substr($e['Names'][0], 1),
			'imageId'      => $e['ImageID'],
			'imageIdShort' => substr($e['ImageID'], 7, 12),
			'labels'       => $e['Labels'],
			'state'        => $e['State'],
			'status'       => $e['Status']
		];
	}

	return $containers;
}

function containerCommand($e, $deploymentName): string {
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
	if (!empty($e['log-opt'])) {
		$params .= ' --log-opt '.$e['log-opt'];
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
	if (!empty($e['volumes'])) {
		$params .= ' -v '.implode(' -v ', $e['volumes']);
	}
	if (!empty($e['environments'])) {
		$params .= ' -e '.implode(' -e ', $e['environments']);
	}

	return 'docker run -d'
		.' --name '.$e['name']
		.' --label '.namespaceDeployment($deploymentName)
		.' '.$params.' '.$e['from'];
}

function selfUpgrade($d) {
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
	echo shell_exec(containerCommand($dc, $d['deployment-name']).' 2>&1').PHP_EOL;

	if (!empty($oldId)) {
		echo '• Container '.$dc['name'].'-old rm force: ';
		echo shell_exec('docker rm -f '.$dc['name'].'-old 2>&1').PHP_EOL;
	}

	exit;
}

function destroy() {
	$ids = explode(PHP_EOL,
		trim(shell_exec('docker ps -a --filter "label='.namespaceDeployment().'" --format "{{.Names}}"'))
	);
	if (!empty($ids)) {
		$spaceCargo = namespaceShip().'.cargo-deployer';
		$ids = array_flip($ids);
		unset($ids[$spaceCargo]);
		$ids = implode(' ', array_flip($ids));

		echo '• Containers destroy: ';
		echo shell_exec('docker rm -f '.$ids.' 2>&1').PHP_EOL;

		echo '• Self destroy: ';
		echo shell_exec('docker rm -f '.$spaceCargo.' 2>&1').PHP_EOL;
	}
	exit;
}

function containersExecute($data) {
	$data = json_decode($data, true);

	if (empty($data)) {
		echo '• Empty data options'.PHP_EOL;

		return;
	}

	foreach ($data as $d) {
		if ($d['self-upgrade']) {
			selfUpgrade($d);
		}

		if (($d['destroy'] ?? false) === true) {
			destroy();
		}

		echo '••••••••••'.PHP_EOL;
		echo '• Execution deployment: '.$d['deployment-name'].PHP_EOL;

		if (!isset($d['containers']))
			$d['containers'] = [];

		foreach ($d['containers'] as $e) {
			echo '• Pulling "'.$e['from'].'": ';
			echo shell_exec('docker pull '.$e['from']).PHP_EOL;
		}

		$ids = str_replace(PHP_EOL, ' ',
			shell_exec('docker ps -aqf "label='.namespaceDeployment($d['deployment-name']).'"')
		);
		if (!empty($ids)) {
			echo '• Containers stop: ';
			echo shell_exec('docker stop '.$ids.' 2>&1').PHP_EOL;
			echo '• Containers rm: ';
			echo shell_exec('docker rm '.$ids.' 2>&1').PHP_EOL;
		}

		foreach ($d['containers'] as $e) {
			echo '• Container run "'.$e['name'].'": ';
			echo shell_exec(containerCommand($e, $d['deployment-name']).' 2>&1').PHP_EOL;

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

		echo '••••••••••'.PHP_EOL;
	}

	echo '• Image prune'.PHP_EOL;
	echo shell_exec('docker image prune -f').PHP_EOL;
}

while (true) {
	$result = push(++$iteration);

	if (($result['ok'] ?? false) === true) {
		if (!empty($result['data']['execs'])) {
			echo '['.date('Y-m-d H:i:s').'] Received data for deployments'.PHP_EOL;
			containersExecute($result['data']['execs']);
			echo '['.date('Y-m-d H:i:s').'] End deployments'.PHP_EOL;

			push($iteration);
		}

	} else {
		echo '['.date('Y-m-d H:i:s').'] '.print_r($result, true).PHP_EOL;
	}

	sleep($timer);
}