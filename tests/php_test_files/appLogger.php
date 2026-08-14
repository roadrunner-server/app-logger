<?php

ini_set('display_errors', 'stderr');
require __DIR__ . "/vendor/autoload.php";

use Spiral\Goridge;
use RoadRunner\Logger\Logger;

$rpc = new Goridge\RPC\RPC(
    Goridge\Relay::create('tcp://127.0.0.1:6301')
);

$logger = new Logger($rpc);

/**
 * debug mapped to RR's debug logger
 */
$logger->debug('Debug message');

/**
 * error mapped to RR's error logger
 */
$logger->error('Error message');

/**
 * log mapped to RR's stderr
 */
$logger->log("Log message \n");

/**
 * info mapped to RR's info logger
 */
$logger->info('Info message');

/**
 * warning mapped to RR's warning logger
 */
$logger->warning('Warning message');
