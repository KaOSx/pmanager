(function (m) {
    var config = {
        repourl: 'https://kaosx.tk/repo/',
        apiurl: 'http://localhost:9000',
        mirrorurl: 'mirrors.html',
        route: {
            repo_list: '/',
            package_list_in_repo: '/repo/:repo',
            package_list: '/list',
            package_view: '/package/:repo/:name',
            package_flag: '/flag/:repo/:name',
            flag_list: '/flagged',
            not_found: '/:404...',
        },
        request: {
            package_list_in_repo: '/repo/list',
            package_list: '/package/list',
            package_view: '/package/view',
            flag_list: '/flag/list?sortby=date&sortdir=desc&flagged=1',
        },
        default_route: 'repo_list',
        incpage: 5,
        repository: {
            apps: 'KDE SC and applications',
            build: 'Build repository for all updates and rebuilds',
            core: 'The minimal stable base of the system',
            'kde-next': 'KDE SC and applications',
            main: 'Main stable deps, add-ons and drivers',
        },
        getRepositoryDescription(repo) {
            if (typeof this.repository[repo] === 'undefined') {
                return '';
            }
            return this.repository[repo];
        },
    };

    var state = {
        route: {
            name: '',
            path: '',
            argspath: '',
            args: {},
        },
        result: {},
        sendRequest: function () {
            this.result = {};
            var sd = config.request[this.route.name];
            if (typeof sd === 'undefined') {
                return;
            }
            var url = config.apiurl + sd;
            var args = m.buildQueryString(this.route.args);
            if (args) {
                url += '?' + args;
            }
            m.request({
                url: url,
                method: 'GET',
                responseType: 'json',
                headers: {
                    'Content-Type': 'application/json',
                },
            })
                .then(function (response) {
                    state.result = response;
                })
                .catch(function (e) {
                    var eargs = m.parseQueryString(args);
                    eargs.err = state.route.path+'?'+state.route.argspath;
                    m.route.set('/404', eargs);
                });
        },
        init: function (name, component) {
            return {
                onmatch: function (args, path) {
                    state.route.name = name;
                    state.route.args = args;
                    var i = path.indexOf('?');
                    if (i < 0) {
                        state.route.path = path;
                        state.route.argspath = '';
                    } else {
                        state.route.path = path.substring(0, i);
                        state.route.argspath = path.substring(i + 1);
                    }
                    state.sendRequest();
                    return component;
                },
            };
        },
    };

    var block = {
        buttons_bar: {
            view: function () {
                return m('div#toolbar.tdrt2', [
                    m('a.btn', { href: config.mirrorurl }, 'Mirror status'),
                    m('a.btn', {
                        href: config.route.flag_list,
                        oncreate: m.route.link,
                        style: { 'margin-left': '4px' },
                    }, 'Flagged'),
                    m('a.btn', {
                        href: config.route.package_list + '?sortby=date&sortdir=desc',
                        oncreate: m.route.link,
                        style: { 'margin-left': '4px' },
                    }, 'Last packages'),
                    m('a.btn', {
                        href: config.route.package_list + '?sortby=name&sortdir=asc',
                        oncreate: m.route.link,
                        style: { 'margin-left': '4px' },
                    }, 'All packages'),
                ]);
            },
        },
        search_form: {
            initFilters: function () {
                var args = state.route.args;
                if (typeof args.sortby === 'undefined') {
                    args.sortby = 'name';
                }
                if (typeof args.sortdir === 'undefined') {
                    args.sortdir = 'asc';
                }
                return args;
            },
            setSearch: function (value) {
                var args = state.route.args;
                args.search = value;
                delete args.page;
            },
            execSearch: function () {
                var route = (state.route.name === 'repo_list') ? config.route.package_list : state.route.path;
                var args = state.route.args;
                delete args.repo;
                m.route.set(route, args);
            },
            title: function (repo) {
                var t1 = (repo) ? 'Repo: ' + repo : 'Repositories list';
                var t2 = (repo && config.repository[repo]) ? config.repository[repo] : '';
                return [m('b', t1), m('br'), t2];
            },
            form: function (args) {
                var value = (k) => args[k] ? args[k] : '';
                return m('form[method="post"][name="searchform"]', [
                    m('input[name="sortby"][type="hidden"]', { value: value('sortby') }),
                    m('input[name="sortdir"][type="hidden"]', { value: value('sortdir') }),
                    m('input[name="flagged"][type="hidden"]', { value: value('flagged') }),
                    m('input[name="page"][type="hidden"]', { value: value('page') }),
                    m('input[name="search"][type="text"][size="20"]', {
                        oninput: m.withAttr('value', this.setSearch),
                        value: value('search'),
                    }),
                    m('button[type="submit"]', {
                        style: { cursor: 'mouse-pointer' },
                        onclick: this.execSearch,
                    }, 'Search'),
                ]);
            },
            view: function () {
                var args = this.initFilters();
                return m('table.stable[border="0"]', m('tbody', m('tr', [
                    m('td', this.title(args.repo)),
                    m('td.tdrt[width="28"]'),
                    m('td.tdrt', this.form(args)),
                ])));
            },
        },
        repo_list: {
            line_repo: function (repo) {
                var url = config.route.package_list_in_repo.replace(':repo', repo);
                var args = '?sortby=name&sortdir=asc';
                return m('p', m('a', {
                    href: url + args,
                    oncreate: m.route.link,
                }, m('b', m('i.fa.fa-bars.fa-lg', ' ' + repo))));
            },
            view: function () {
                return Object.keys(config.repository).map(this.line_repo);
            },
        },
        package_list_header: {
            linkColumn: function (name) {
                var args = m.parseQueryString(state.route.argspath);
                if (args.sortby === name) {
                    args.sortdir = (args.sortdir === 'desc') ? 'asc' : 'desc';
                }
                args.sortby = name;
                return state.route.path + '?' + m.buildQueryString(args);
            },
            column: function (name, sortby) {
                return m('th', m('a', {
                    href: this.linkColumn(sortby),
                    oncreate: m.route.link,
                    onupdate: m.route.link,
                }, name));
            },
            view: function () {
                return m('tr', [
                    this.column('Name', 'name'),
                    this.column('Repo', 'repo'),
                    this.column('Size', 'size'),
                    this.column('Date', 'date'),
                    m('th'),
                ]);
            },
        },
        package_list_subheader: {
            view: function () {
                return m('tr', [
                    m('td[colspan="5"]', m('a', {
                        href: config.route.repo_list,
                        oncreate: m.route.link,
                    }, m('b', m('i.fa.fa-bars.fa-lg', ' [Repositories list]')))),
                ]);
            },
        },
        package_list_content: {
            line: function (pack) {
                return m('tr', [
                    m('td', m('a', {
                        href: config.route.package_view.replace(':repo', pack.Repository).replace(':name', pack.CompleteName),
                        oncreate: m.route.link,
                        onupdate: m.route.link,
                    }, pack.CompleteName)),
                    m('td[align="left"]', pack.Repository),
                    m('td[align="right"]', pack.PackageSize),
                    m('td[align="right"]', pack.BuildDate),
                    m('td[align="center"]', m('a', {
                        href: config.repourl + pack.Repository + '/' + pack.FileName,
                    }, m('i.fa.fa-linux.fa-lg'))),
                ]);
            },
            view: function () {
                var data = state.result.data;
                if (data) {
                    return data.map(this.line);
                }
                return '';
            },
        },
        package_list_footer: {
            title: function () {
                var results = state.result;
                var nb = 0, size = 0;
                if (results) {
                    if (results.paginate) {
                        nb = results.paginate.total;
                    }
                    if (results.size) {
                        size = results.size;
                    }
                }
                return 'Total: ' + nb + ' packages (' + size + ')';
            },
            view: function () {
                return m('tr', m('th[colspan="5"]', this.title()));
            },
        },
        package_list: {
            view: function () {
                return m('table.ctable[border="0"][cellspacing="10"][cellpadding="2"]', m('tbody', [
                    m(block.package_list_header),
                    m(block.package_list_subheader),
                    m(block.package_list_content),
                    m(block.package_list_footer),
                ]));
            },
        },
        pagination: {
            setPageParameters: function (args, page) {
                var oargs = m.parseQueryString(args);
                oargs.page = page;
                return m.buildQueryString(oargs);
            },
            paginate: function (current, last) {
                var pages = [];
                var url = state.route.path;
                var args = state.route.argspath;
                if (last != 1) {
                    var prev = current - config.incpage, next = current + config.incpage;
                    if (prev < 1) {
                        prev = 0;
                    }
                    if (next > last) {
                        next = last + 1;
                    }
                    if (prev > 0) {
                        pages.push(m('a', {
                            href: url + '?' + this.setPageParameters(args, 1),
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, '1'));
                    }
                    if (prev > 1) {
                        pages.push(m('a', {
                            href: url + '?' + this.setPageParameters(args, prev),
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, '…'));
                    }
                    for (var i = prev + 1; i < next; i++) {
                        pages.push(m('a', {
                            href: url + '?' + this.setPageParameters(args, i),
                            disabled: i == current,
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, i));
                    }
                    if (next < last) {
                        pages.push(m('a', {
                            href: url + '?' + this.setPageParameters(args, next),
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, '…'));
                    }
                    if (next < last + 1) {
                        pages.push(m('a', {
                            href: url + '?' + this.setPageParameters(args, last),
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, last));
                    }
                }
                return pages;
            },
            view: function () {
                var pager = state.result.paginate;
                var current = 1, last = 1;
                if (pager) {
                    current = pager.page;
                    last = pager.last;
                }
                return m('div#paginate', this.paginate(current, last));
            },
        },
    };

    var controller = {
        repo_list: {
            view: function () {
                return m('#wrapper', [
                    m(block.buttons_bar),
                    m(block.search_form),
                    m('p'),
                    m(block.repo_list),
                ]);
            },
        },
        package_list_in_repo: {
            view: function () {
                return m('#wrapper', [
                    m(block.buttons_bar),
                    m(block.search_form),
                    m(block.package_list),
                    m(block.pagination),
                ]);
            },
        },
        package_list: {
            view: function () {
                return m('#wrapper', [
                    m(block.buttons_bar),
                    m(block.search_form),
                    m(block.package_list),
                    m(block.pagination),
                ]);
            },
        },
        package_view: {
            packageInfos: function (pack) {
                var groups = '';
                if (pack.Groups) {
                    groups = pack.Groups.join(' ');
                }
                var flagurl = config.route.package_flag
                    .replace(':repo', pack.Repository)
                    .replace(':name', pack.CompleteName);
                var out = [
                    m('tr', m('th.tdhp', m('b', pack.CompleteName))),
                    m('tr', m('td', 'Repository: ' + pack.Repository)),
                    m('tr', m('td', 'Description: ' + pack.Description)),
                    m('tr', m('td', [
                        'Upstream URL: ',
                        m('a', { href: pack.URL.Upstream }, pack.URL.Upstream),
                    ])),
                    m('tr', m('td', 'License: ' + pack.Licenses.join(', '))),
                    m('tr', m('td', 'Package size: ' + pack.PackageSize)),
                    m('tr', m('td', 'Installed size: ' + pack.InstalledSize)),
                    m('tr', m('td', 'Build date: ' + pack.BuildDate)),
                    m('tr', m('td', 'Packages groups: [' + groups + ']')),
                    m('tr', m('td')),
                    m('tr', m('td')),
                ];
                if (!pack.Flagged) {
                    if (!['build', 'kde-next'].includes(pack.Repository)) {
                        out.push(m('tr', m('td[align="center"]', m('div.Button', m('a', {
                            href: flagurl,
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, 'Flag as outdated')))))
                    };
                } else {
                    out.push(
                        m('tr', m('td.pkgwarning', m('i.fa.fa-flag-checkered.text-danger', ' This package has been flagged as outdated'))));
                }
                out.push(m('tr', m('td')));
                if (pack.Depends) {
                    out.push(m('tr', m('th.tdhp', m('b', 'Dependencies'))));
                    for (var i = 0; i < pack.Depends.length; i++) {
                        out.push(m('tr', m('td', m('a', {
                            href: config.route.package_list + '?exact=1&search=' + pack.Depends[i],
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, pack.Depends[i]))));
                    }
                }
                if (pack.OptDepends) {
                    out.push(m('tr', m('th.tdhp', m('b', 'Optdepends'))));
                    for (var i = 0; i < pack.OptDepends.length; i++) {
                        out.push(m('tr', m('td', m('a', {
                            href: config.route.package_list + '?exact=1&search=' + pack.OptDepends[i],
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, pack.OptDepends[i]))));
                    }
                }
                if (pack.MakeDepends) {
                    out.push(m('tr', m('th.tdhp', m('b', 'Makedepends'))));
                    for (var i = 0; i < pack.MakeDepends.length; i++) {
                        out.push(m('tr', m('td', m('a', {
                            href: config.route.package_list + '?exact=1&search=' + pack.MakeDepends[i],
                            oncreate: m.route.link,
                            onupdate: m.route.link,
                        }, pack.MakeDepends[i]))));
                    }
                }
                if (pack.Files) {
                    out.push(m('tr', m('th.tdhp', m('b', 'Files listing'))));
                    pack.Files.forEach(f => out.push((m('tr', m('td', f)))));
                }
                var packageurl = config.route.package_list_in_repo.replace(':repo', pack.Repository);
                out.push(m('tr', m('td[align="center"]', m('div.Button', m('a', {
                    href: packageurl + '?sortby=date&sortdir=desc',
                    oncreate: m.route.link,
                    onupdate: m.route.link,
                }, 'Return to packages')))));
                return out;
            },
            gitInfos: function (pack) {
                var url = pack.URL;
                return [
                    m('h3', [
                        m('i.fa.fa-download'),
                        ' Package',
                    ]),
                    m('ul.pkglinks', [
                        m('li', m('a[href="' + url.Download + '"]', 'Download')),
                    ]),
                    m('h3', [
                        m('i.fa.fa-bug'),
                        ' Bugs',
                    ]),
                    m('ul.pkglinks', [
                        m('li', m('a[href="' + url.Bugs + '"]', 'Report Issues')),
                    ]),
                    m('h3', [
                        m('i.fa.fa-github-square'),
                        ' Git',
                    ]),
                    m('ul.pkglinks', [
                        m('li', m('a[href="' + url.Sources + '"]', 'Source Files')),
                        m('li', m('a[href="' + url.PKGBUILD + '"]', 'PKGBUILD')),
                        m('li', m('a[href="' + url.Commits + '"]', 'Commits')),
                    ]),
                ];
            },
            view: function () {
                if (state.result.data) {
                    return m('#wrapper', [
                        m('p'),
                        m('br'),
                        m('div.wrapper', m('div#linkList', m('div#linkList2', this.gitInfos(state.result.data)))),
                        m('table.ltable[width="60%"][border="0"][cellspacing="10"][cellpadding="2"]', m('tbody', this.packageInfos(state.result.data))),
                    ]);
                }
                return '';
            },
        },
        package_flag: {
            setEmail: function (value) {
                state.email = value;
            },
            setComment: function (value) {
                value = value.replace(/\r\n/g, '\n').replace(/<br>/g, '\n');
                state.comment = value.replace(/<[^>]*>/g, '');
            },
            oninit: function () {
                state.package = {};
                var sd = config.request.package_view;
                var url = config.apiurl + sd;
                var args = m.buildQueryString(state.route.args);
                if (args) {
                    url += '?' + args;
                }
                m.request({
                    url: url,
                    method: 'GET',
                    responseType: 'json',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                })
                    .then(function (response) {
                        state.package = response.data;
                    })
                    .catch(function (e) {
                        var eargs = m.parseQueryString(args);
                        eargs.err = state.route.path+'?'+state.route.argspath;
                        m.route.set('/404', eargs);
                    });
            },
            submitFlag: function (e) {
                e.preventDefault();
                if (state.submitted) return;
                var url = config.apiurl + '/flag/add';
                var args = {
                    repo: state.package.Repository,
                    name: state.package.Name,
                    version: state.package.Version,
                    email: state.email,
                    comment: state.comment,
                };
                args = m.buildQueryString(args);
                url += '?' + args;
                m.request({
                    url: url,
                    method: 'GET',
                    responseType: 'json',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                }).then(function (response) {
                    state.submitted = true;
                    var url = config.route.package_view
                        .replace(':repo', state.package.Repository)
                        .replace(':name', state.package.CompleteName);
                    m.route.set(url);
                })
                    .catch(function (e) {
                        var eargs = m.parseQueryString(args);
                        eargs.err = state.route.path+'?'+state.route.argspath;
                        m.route.set('/404', eargs);
                    });
            },
            viewUnsubmitted() {
                var pack = state.package;
                if (!pack) {
                    return '';
                }
                if (pack.Flagged) {
                    return m('p', [
                        'The package ',
                        m('b', pack.CompleteName),
                        ' is already flagged!',
                    ]);
                }
                if (['build', 'kde-next'].includes(pack.Repository)) {
                    return m('p', [
                        'You cannot submit a flag on a test package',
                    ]);
                }
                return [
                    m('form[method="post"]', {
                        onsubmit: this.submitFlag,
                    }, m('table.cctable[border="0"]', m('tbody', [
                        m('tr', m('td.cctable', [
                            'Your email: ',
                            m('input[type="email"][required="required"][size="50"][name="email"]', {
                                value: state.email,
                                oninput: m.withAttr('value', this.setEmail),
                            }),
                            m('br'),
                            m('br'),
                            m('br'),
                        ])),
                        m('tr', m('td.cctable', [
                            'You are about to flag ',
                            m('b', pack.CompleteName),
                            ' as outdated, write any additional information here.',
                            m('br'),
                            'Use ',
                            m('a[href="https://kaosx.us/bugs"]', m('u', 'Bugs')),
                            ' ',
                            m('b', 'for broken packages'),
                            '.',
                            m('br'),
                            m('br'),
                            m('textarea[cols="50"][required="required"][rows="8"][name="comment"]', {
                                value: state.comment,
                                oninput: m.withAttr('value', this.setComment),
                            }),
                            m('br'),
                            m('br'),
                        ])),
                        m('tr', m('td.cctable', [
                            m('input[type="checkbox"][name="gdpr"][required="required"]'),
                            ' I authorize KaOS to record my email',
                        ])),
                        m('tr', m('td[align="center"]', [
                            m('br'),
                            m('br'),
                            m('br'),
                            m('button[type="submit"]', 'Flag the package as outdated'),
                        ])),
                    ]))),
                    m('br'),
                    m('br'),
                ];
            },
            viewSubmitted() {
                return '';
            },
            view: function () {
                return m('#wrapper', [
                    state.submitted ? this.viewSubmitted() : this.viewUnsubmitted(),
                    m('div.Button[align="center"]', m('a', {
                        href: config.route.package_view.replace(':repo', state.package.Repository).replace(':name', state.package.CompleteName),
                        oncreate: m.route.link,
                        onupdate: m.route.link,
                    }, 'Back to package')),
                ]);
            },
        },
        flag_list: {
            list: function () {
                if (!state.result.data) {
                    return '';
                }
                return state.result.data.map(function (f) {
                    var line = [
                        m('b', [
                            f.Name,
                            ' - ',
                            f.Version,
                        ]),
                        f.Flagged ? ' (outdated)' : '',
                        ' :',
                        m('br'),
                    ];
                    f.Comment.split('\n').forEach(c => line.push(c, m('br')));
                    return m('p', line);
                });
            },
            view: function () {
                return m('#wrapper', m('.ctable', m('.fltable[align="left"]', [
                    m('.line', m('h4', 'Flagged packages')),
                    m('.fltable', this.list()),
                    m('.Button', m('a', {
                        href: config.route[config.default_route],
                        oncreate: m.route.link,
                    }, 'Return to search index page')),
                ])));
            },
        },
        not_found: {
            view: function () {
              var badroute = state.route.args.err;
              if (!badroute) {
                badroute = state.route.path;
              }
              return m('p', 'No route found for '+badroute);
            },
        },
    };

    var routes = {};
    Object.keys(config.route).forEach(function (k) {
        routes[config.route[k]] = state.init(k, controller[k]);
    });

    const root = document.querySelector('#content');

    window.package_viewer = {
        root: root,
        config: config,
        state: state,
        routes: routes,
        controller: controller,
        block: block,
    };

    m.route(root, config.route[config.default_route], routes);
})(m);
