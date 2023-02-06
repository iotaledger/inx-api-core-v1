const path = require('path');

module.exports = {
    plugins: [
        [
            '@docusaurus/plugin-content-docs',
            {
                id: 'inx-api-core-v1',
                path: path.resolve(__dirname, 'docs'),
                routeBasePath: 'inx-api-core-v1',
                sidebarPath: path.resolve(__dirname, 'sidebars.js'),
                editUrl: 'https://github.com/iotaledger/inx-api-core-v1/edit/develop/documentation',
            }
        ],
    ],
    staticDirectories: [path.resolve(__dirname, 'static')],
};
