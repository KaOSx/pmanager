<?php

include __DIR__.'/inc/vars.php';
include __DIR__.'/inc/util.php';

$page_title = 'Online Package Viewer';

function render()
{
    $repo = isset($_GET['repo']) ? $_GET['repo'] : '';
    $name = isset($_GET['name']) ? $_GET['name'] : '';
    $result = execRequest('/package/view', [
        'repo' => $repo,
        'name' => $name,
    ]);
    if (!$result) {
        echo 'Package “'.$repo.'/'.$name.'” not found.';
    }
};

include __DIR__.'/tpl/page.php';