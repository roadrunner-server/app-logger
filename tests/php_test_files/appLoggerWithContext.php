<?php

ini_set('display_errors', 'stderr');
require __DIR__ . "/vendor/autoload.php";

use Spiral\Goridge;
use RoadRunner\Logger\Logger;

$rpc = new Goridge\RPC\RPC(
    Goridge\Relay::create('tcp://127.0.0.1:6302')
);

$logger = new Logger($rpc);

/**
 * debug with context attributes
 */
$logger->debug('Debug context message', ['component' => 'test', 'request_id' => '12345']);

/**
 * error with context attributes
 */
$logger->error('Error context message', ['error_code' => '500', 'trace' => 'stack_trace_here']);

/**
 * info with context attributes
 */
$logger->info('Info context message', ['user' => 'john']);

/**
 * warning with context attributes
 */
$logger->warning('Warning context message', ['threshold' => '90']);

/**
 * log with context attributes (writes to stderr)
 */
$logger->log("Log context message\n", ['source' => 'worker']);
