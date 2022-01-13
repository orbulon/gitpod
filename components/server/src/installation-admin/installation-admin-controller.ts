/**
 * Copyright (c) 2022 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { injectable } from 'inversify';
import * as express from 'express';

@injectable()
export class InstallationAdminController {
    get apiRouter(): express.Router {
        const router = express.Router();

        router.get('/data', async (req: express.Request, res: express.Response) => {
            res.send({
                sendTelemetry: false
            });
        });

        return router;
    }
}
