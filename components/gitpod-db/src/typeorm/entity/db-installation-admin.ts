/**
 * Copyright (c) 2021 Gitpod GmbH. All rights reserved.
 * Licensed under the GNU Affero General Public License (AGPL).
 * See License-AGPL.txt in the project root for license information.
 */

import { InstallationAdmin } from "@gitpod/gitpod-protocol";
import { Entity, Column, PrimaryColumn, CreateDateColumn, UpdateDateColumn } from "typeorm";
import { TypeORM } from "../typeorm";

@Entity()
export class DBInstallationAdmin implements InstallationAdmin {
  @PrimaryColumn(TypeORM.UUID_COLUMN_TYPE)
  id: string;

  @Column()
  sendTelemetry: boolean

  @CreateDateColumn({
    type: 'datetime',
  })
  createdAt: Date;

  @UpdateDateColumn({
    type: 'datetime',
  })
  updatedAt: Date;
}