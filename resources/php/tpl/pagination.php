<div id="paginate">
    <?php foreach ($pages as $p): ?>
    <a href="<?= $p['url'] ?>" <?= ($p['Current']) ? 'disabled' : '' ?>>
        <?= $p['text'] ?>
    </a>
    <?php endforeach; ?>
</div>