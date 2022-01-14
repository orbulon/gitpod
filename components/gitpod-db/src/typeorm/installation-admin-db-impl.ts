/**
 * Copyright (c) 2022 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { inject, injectable, } from 'inversify';
import { TypeORM } from './typeorm';
import { InstallationAdminDB } from '../installation-admin-db'

@injectable()
export class TypeORMInstallationAdminImpl implements InstallationAdminDB {
    @inject(TypeORM) typeORM: TypeORM;

    protected async getEntityManager() {
        return (await this.typeORM.getConnection()).manager;
    }
}
