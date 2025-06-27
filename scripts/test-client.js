#!/usr/bin/env node

/**
 * Script per testare il client TypeScript generato
 * Usage: node scripts/test-client.js
 */

const fs = require("fs");
const path = require("path");

// Colori per output
const colors = {
	red: "\x1b[31m",
	green: "\x1b[32m",
	yellow: "\x1b[33m",
	blue: "\x1b[34m",
	reset: "\x1b[0m",
};

function log(color, message) {
	console.log(`${colors[color]}${message}${colors.reset}`);
}

function testClientStructure() {
	log("blue", "🔍 Testando struttura del client...");

	const clientDir = "generated-client";
	const requiredFiles = [
		"package.json",
		"tsconfig.json",
		"index.ts",
		"runtime.ts",
		"api/agents-api.ts",
		"api/participants-api.ts",
		"api/services-api.ts",
		"api/jobs-api.ts",
		"api/events-api.ts",
		"api/metrics-api.ts",
		"api/tokens-api.ts",
		"models/agent.ts",
		"models/participant.ts",
		"models/service.ts",
		"models/job.ts",
		"models/event.ts",
		"models/token.ts",
	];

	let allFilesExist = true;

	for (const file of requiredFiles) {
		const filePath = path.join(clientDir, file);
		if (fs.existsSync(filePath)) {
			log("green", `✅ ${file}`);
		} else {
			log("red", `❌ ${file} - MANCANTE`);
			allFilesExist = false;
		}
	}

	return allFilesExist;
}

function testPackageJson() {
	log("blue", "\n📦 Testando package.json...");

	try {
		const packagePath = path.join("generated-client", "package.json");
		const packageJson = JSON.parse(fs.readFileSync(packagePath, "utf8"));

		const requiredFields = ["name", "version", "description", "main", "types"];
		let allFieldsExist = true;

		for (const field of requiredFields) {
			if (packageJson[field]) {
				log("green", `✅ ${field}: ${packageJson[field]}`);
			} else {
				log("red", `❌ ${field} - MANCANTE`);
				allFieldsExist = false;
			}
		}

		// Verifica script
		if (packageJson.scripts && packageJson.scripts.build) {
			log("green", "✅ Script build presente");
		} else {
			log("red", "❌ Script build mancante");
			allFieldsExist = false;
		}

		return allFieldsExist;
	} catch (error) {
		log("red", `❌ Errore nel leggere package.json: ${error.message}`);
		return false;
	}
}

function testTypeScriptConfig() {
	log("blue", "\n⚙️  Testando tsconfig.json...");

	try {
		const tsConfigPath = path.join("generated-client", "tsconfig.json");
		const tsConfig = JSON.parse(fs.readFileSync(tsConfigPath, "utf8"));

		const requiredFields = ["compilerOptions", "include", "exclude"];
		let allFieldsExist = true;

		for (const field of requiredFields) {
			if (tsConfig[field]) {
				log("green", `✅ ${field} presente`);
			} else {
				log("red", `❌ ${field} - MANCANTE`);
				allFieldsExist = false;
			}
		}

		// Verifica configurazioni TypeScript
		if (tsConfig.compilerOptions) {
			const requiredOptions = ["target", "module", "declaration", "outDir"];
			for (const option of requiredOptions) {
				if (tsConfig.compilerOptions[option]) {
					log(
						"green",
						`✅ compilerOptions.${option}: ${tsConfig.compilerOptions[option]}`
					);
				} else {
					log("yellow", `⚠️  compilerOptions.${option} - MANCANTE`);
				}
			}
		}

		return allFieldsExist;
	} catch (error) {
		log("red", `❌ Errore nel leggere tsconfig.json: ${error.message}`);
		return false;
	}
}

function testApiFiles() {
	log("blue", "\n🔌 Testando file API...");

	const apiDir = path.join("generated-client", "api");
	const apiFiles = [
		"agents-api.ts",
		"participants-api.ts",
		"services-api.ts",
		"jobs-api.ts",
		"events-api.ts",
		"metrics-api.ts",
		"tokens-api.ts",
	];

	let allApisValid = true;

	for (const apiFile of apiFiles) {
		const filePath = path.join(apiDir, apiFile);
		if (fs.existsSync(filePath)) {
			const content = fs.readFileSync(filePath, "utf8");

			// Verifica che il file contenga una classe API
			if (content.includes("export class") && content.includes("Api")) {
				log("green", `✅ ${apiFile} - Classe API valida`);
			} else {
				log("red", `❌ ${apiFile} - Classe API non trovata`);
				allApisValid = false;
			}
		} else {
			log("red", `❌ ${apiFile} - FILE MANCANTE`);
			allApisValid = false;
		}
	}

	return allApisValid;
}

function testModelFiles() {
	log("blue", "\n📋 Testando file modelli...");

	const modelsDir = path.join("generated-client", "models");
	const modelFiles = [
		"agent.ts",
		"participant.ts",
		"service.ts",
		"job.ts",
		"event.ts",
		"token.ts",
	];

	let allModelsValid = true;

	for (const modelFile of modelFiles) {
		const filePath = path.join(modelsDir, modelFile);
		if (fs.existsSync(filePath)) {
			const content = fs.readFileSync(filePath, "utf8");

			// Verifica che il file contenga interfacce o tipi
			if (
				content.includes("export interface") ||
				content.includes("export type")
			) {
				log("green", `✅ ${modelFile} - Interfaccia/Tipo valido`);
			} else {
				log("red", `❌ ${modelFile} - Interfaccia/Tipo non trovato`);
				allModelsValid = false;
			}
		} else {
			log("red", `❌ ${modelFile} - FILE MANCANTE`);
			allModelsValid = false;
		}
	}

	return allModelsValid;
}

function testBuild() {
	log("blue", "\n🔨 Testando build...");

	try {
		const { execSync } = require("child_process");
		execSync("npm run build", {
			cwd: "generated-client",
			stdio: "pipe",
		});
		log("green", "✅ Build completata con successo");
		return true;
	} catch (error) {
		log("red", `❌ Build fallita: ${error.message}`);
		return false;
	}
}

function main() {
	log("blue", "🚀 Avvio test del client TypeScript generato\n");

	const tests = [
		{ name: "Struttura file", test: testClientStructure },
		{ name: "Package.json", test: testPackageJson },
		{ name: "TypeScript config", test: testTypeScriptConfig },
		{ name: "File API", test: testApiFiles },
		{ name: "File modelli", test: testModelFiles },
		{ name: "Build", test: testBuild },
	];

	let allTestsPassed = true;

	for (const { name, test } of tests) {
		try {
			const result = test();
			if (!result) {
				allTestsPassed = false;
			}
		} catch (error) {
			log("red", `❌ Test "${name}" fallito: ${error.message}`);
			allTestsPassed = false;
		}
	}

	log("blue", "\n📊 Risultati test:");
	if (allTestsPassed) {
		log("green", "🎉 Tutti i test sono passati! Il client è pronto per l'uso.");
	} else {
		log(
			"red",
			"❌ Alcuni test sono falliti. Controlla la generazione del client."
		);
		process.exit(1);
	}
}

// Esegui test solo se il client esiste
if (fs.existsSync("generated-client")) {
	main();
} else {
	log(
		"red",
		"❌ Directory generated-client non trovata. Esegui prima la generazione del client."
	);
	log("yellow", "💡 Usa: npm run generate-client:test");
	process.exit(1);
}
