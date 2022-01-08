// Copyright (c) 2021 Gitpod GmbH. All rights reserved.
// Licensed under the GNU Affero General Public License (AGPL).
// See License-AGPL.txt in the project root for license information.

package io.gitpod.jetbrains.remote.services

import com.intellij.ide.plugins.PluginManagerCore
import com.intellij.openapi.Disposable
import com.intellij.openapi.components.Service
import com.intellij.openapi.diagnostic.thisLogger
import com.intellij.openapi.extensions.PluginId
import io.gitpod.gitpodprotocol.api.GitpodClient
import io.gitpod.gitpodprotocol.api.GitpodServerLauncher
import io.gitpod.gitpodprotocol.api.entities.SendHeartBeatOptions
import io.gitpod.jetbrains.remote.services.ControllerStatusService.ControllerStatus
import kotlinx.coroutines.GlobalScope
import kotlinx.coroutines.delay
import kotlinx.coroutines.future.await
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import javax.websocket.DeploymentException
import kotlin.coroutines.coroutineContext
import kotlin.random.Random.Default.nextInt

@Service
class HeartbeatService : Disposable {

    private val job = GlobalScope.launch {
        val info = SupervisorInfoService.fetch()
        val client = GitpodClient()
        val launcher = GitpodServerLauncher.create(client)
        launch {
            connectToServer(info, launcher)
        }
        val intervalInSeconds = 30
        var current = ControllerStatus(
            connected = false,
            secondsSinceLastActivity = 0
        )
        while (isActive) {
            try {
                val previous = current;
                current = ControllerStatusService.fetch()

                val maxIntervalInSeconds = intervalInSeconds + nextInt(5, 15)
                val wasClosed: Boolean? = when {
                    current.connected != previous.connected -> !current.connected
                    current.connected && current.secondsSinceLastActivity <= maxIntervalInSeconds -> false
                    else -> null
                }

                if (wasClosed != null) {
                    client.server.sendHeartBeat(SendHeartBeatOptions(info.instanceId, wasClosed)).await()
                }
            } catch (t: Throwable) {
                thisLogger().error("gitpod: failed to check activity:", t)
            }
            delay(intervalInSeconds * 1000L)
        }
    }

    private suspend fun connectToServer(info: SupervisorInfoService.Info, launcher: GitpodServerLauncher) {
        val plugin = PluginManagerCore.getPlugin(PluginId.getId("io.gitpod.jetbrains.remote"))!!
        val connect = {
            val originalClassLoader = Thread.currentThread().contextClassLoader
            try {
                // see https://intellij-support.jetbrains.com/hc/en-us/community/posts/360003146180/comments/360000376240
                Thread.currentThread().contextClassLoader = HeartbeatService::class.java.classLoader
                launcher.listen(
                    "wss://${info.host.split("//").last()}/api/v1",
                    info.workspaceUrl,
                    plugin.pluginId.idString,
                    plugin.version,
                    info.authToken
                )
            } finally {
                Thread.currentThread().contextClassLoader = originalClassLoader;
            }
        }

        val minReconnectionDelay = 2 * 1000L
        val maxReconnectionDelay = 30 * 1000L
        val reconnectionDelayGrowFactor = 1.5;
        var reconnectionDelay = minReconnectionDelay;
        while (coroutineContext.isActive) {
            try {
                val connection = connect()
                reconnectionDelay = minReconnectionDelay
                val reason = connection.await()
                if (coroutineContext.isActive) {
                    thisLogger().warn("gitpod server: connection closed, reconnecting after $reconnectionDelay milliseconds: $reason")
                } else {
                    thisLogger().info("gitpod server: connection permanently closed: $reason")
                }
            } catch (e: DeploymentException) {
                thisLogger().error("gitpod server: failed to establish connection:", e)
                return;
            } catch (t: Throwable) {
                if (coroutineContext.isActive) {
                    thisLogger().warn(
                        "gitpod server: failed to connect, trying again after $reconnectionDelay milliseconds:",
                        t
                    )
                } else {
                    thisLogger().error("gitpod server: connection permanently closed: $t")
                }
            }
            delay(reconnectionDelay)
            reconnectionDelay = (reconnectionDelay * reconnectionDelayGrowFactor).toLong()
            if (reconnectionDelay > maxReconnectionDelay) {
                reconnectionDelay = maxReconnectionDelay
            }
        }
    }

    override fun dispose() = job.cancel()
}
