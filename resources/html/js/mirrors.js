(function (m) {
    var config = {
        apiurl: 'http://localhost:9000',
        request: '/mirror',
        indexpage: 'packages.html',
    };

    result: [];

    var viewMirrorLine = function (mirror) {
        var l = mirror.Repos.length;
        var c = 0;
        mirror.Repos.forEach(function (r) { if (r.Sync) c++; });
        return [
            m('a', { href: mirror.Name }, mirror.Name),
            ' | ',
            m('font', { color: mirror.Online ? 'green' : 'red' }, mirror.Online ? 'Online' : 'Offline'),
            ' | ',
            m('font', { color: (c == 0) ? 'red' : ((c == l) ? 'green' : 'purple') }, (c == 0) ? 'Not synced' : ((c == l) ? 'Fully synced' : 'Partially synced')),
            m('br'),
        ];
    };

    var viewCountryLine = function (country) {
        var lines = [
            m('b', country.Name),
            m('br'),
        ];
        var addAll = (a) => a.forEach((e) => lines.push(e));
        country.Mirrors.forEach((mirror) => addAll(viewMirrorLine(mirror)));
        return m('p', lines);
    };

    var viewResult = function (results) {
        return m('#wrapper', [
            m('h4', 'Status report of all mirrors used by KaOS.'),
            m('.box', results.map(viewCountryLine)),
            m('.Button', m('a', {
                href: config.indexpage,
            }, 'Return to search index page ')),
        ]);
    };

    const root = document.querySelector('#content');

    var sendRequest = function () {
        var sd = config.request;
        if (typeof sd === 'undefined') {
            return;
        }
        var url = config.apiurl + sd;
        m.request({
            url: url,
            method: 'GET',
            responseType: 'json',
            headers: {
                'Content-Type': 'application/json',
            },
        })
            .then(function (response) {
                results = response;
                m.render(root, viewResult(results));
            });
    };

    sendRequest();
})(m);
